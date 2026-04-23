// Package sandbox provides a hardened wrapper around os/exec for running
// untrusted external tools (protoc, language plugins, code generators, etc.)
// with timeouts, output bounds, environment whitelisting and path validation.
//
// The goal is not full OS-level isolation (that requires containers or
// seccomp), but to remove the most common foot-guns when invoking third-party
// binaries from a build pipeline:
//
//   - All commands run via exec.CommandContext so a canceled parent context
//     terminates the child.
//   - WaitDelay caps how long a process may keep stdio open after the context
//     is canceled (Go 1.20+).
//   - Output is captured into bounded buffers; runaway tools cannot exhaust
//     memory.
//   - The process environment is built from an explicit whitelist instead of
//     leaking the entire parent environment.
//   - When AllowedRoots is non-empty, every path-shaped argument must resolve
//     inside one of the allowed roots; this prevents a malicious config from
//     making protoc read or write arbitrary files.
package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/massonsky/buffalo/pkg/errors"
)

// DefaultTimeout is applied when Options.Timeout is zero.
const DefaultTimeout = 5 * time.Minute

// DefaultWaitDelay bounds the grace period after context cancellation before
// the process is forcibly killed.
const DefaultWaitDelay = 5 * time.Second

// DefaultMaxOutputBytes bounds combined stdout+stderr capture per stream.
const DefaultMaxOutputBytes = 16 * 1024 * 1024 // 16 MiB

// Options controls a single sandboxed execution.
type Options struct {
	// Name is the program name or absolute path. Required.
	Name string

	// Args are the arguments passed to the program (excluding argv[0]).
	Args []string

	// Dir is the working directory. Empty uses the current process CWD.
	Dir string

	// Env is the explicit environment for the child process. When nil, a
	// minimal whitelisted environment is built from os.Environ() (see
	// MinimalEnv). Use ExtraEnv to add custom keys without losing the
	// whitelist.
	Env []string

	// ExtraEnv adds KEY=VALUE entries on top of the whitelisted environment.
	// Ignored when Env is non-nil.
	ExtraEnv []string

	// Stdin is the optional input stream piped to the child.
	Stdin io.Reader

	// Timeout caps total wall-clock execution time. Zero means DefaultTimeout.
	Timeout time.Duration

	// WaitDelay bounds the grace period after the context is canceled before
	// the child is killed. Zero means DefaultWaitDelay.
	WaitDelay time.Duration

	// MaxOutputBytes bounds captured stdout and stderr each. Zero means
	// DefaultMaxOutputBytes. Negative disables the bound (not recommended).
	MaxOutputBytes int64

	// AllowedRoots, when non-empty, restricts path-shaped arguments to be
	// inside one of these directories. Each root must be an absolute path or
	// will be made absolute relative to Dir (or process CWD).
	AllowedRoots []string
}

// Result captures the outcome of an execution.
type Result struct {
	Stdout      []byte
	Stderr      []byte
	ExitCode    int
	StdoutTrunc bool
	StderrTrunc bool
	Duration    time.Duration
	TimedOut    bool
}

// Run executes the command described by opts. The returned error is non-nil
// for non-zero exit codes, timeouts, validation failures and IO errors. The
// Result is always returned (possibly partial) so callers can surface stderr.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if opts.Name == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "sandbox: command name is required")
	}

	if err := validateArgs(opts); err != nil {
		return nil, err
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, opts.Name, opts.Args...)
	cmd.Dir = opts.Dir
	cmd.WaitDelay = opts.WaitDelay
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = DefaultWaitDelay
	}

	if opts.Env != nil {
		cmd.Env = opts.Env
	} else {
		cmd.Env = append(MinimalEnv(), opts.ExtraEnv...)
	}

	limit := opts.MaxOutputBytes
	if limit == 0 {
		limit = DefaultMaxOutputBytes
	}
	stdout := newBoundedBuffer(limit)
	stderr := newBoundedBuffer(limit)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = opts.Stdin

	start := time.Now()
	runErr := cmd.Run()
	dur := time.Since(start)

	res := &Result{
		Stdout:      stdout.Bytes(),
		Stderr:      stderr.Bytes(),
		StdoutTrunc: stdout.truncated,
		StderrTrunc: stderr.truncated,
		Duration:    dur,
		TimedOut:    cctx.Err() == context.DeadlineExceeded,
	}
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	}

	if res.TimedOut {
		return res, errors.Wrap(runErr, errors.ErrTimeout,
			"sandbox: %s timed out after %s", opts.Name, timeout)
	}
	if runErr != nil {
		return res, errors.Wrap(runErr, errors.ErrInternal,
			"sandbox: %s exited with code %d", opts.Name, res.ExitCode)
	}
	return res, nil
}

// validateArgs enforces AllowedRoots containment for path-shaped arguments.
func validateArgs(opts Options) error {
	if len(opts.AllowedRoots) == 0 {
		return nil
	}

	roots := make([]string, 0, len(opts.AllowedRoots))
	for _, r := range opts.AllowedRoots {
		abs, err := absPath(r, opts.Dir)
		if err != nil {
			return errors.Wrap(err, errors.ErrInvalidArgument,
				"sandbox: invalid allowed root %q", r)
		}
		roots = append(roots, abs)
	}

	for _, raw := range opts.Args {
		p := extractPath(raw)
		if p == "" {
			continue
		}
		abs, err := absPath(p, opts.Dir)
		if err != nil {
			// non-path-looking argument; ignore.
			continue
		}
		if !insideAny(abs, roots) {
			return errors.New(errors.ErrInvalidArgument,
				"sandbox: argument %q resolves outside allowed roots", raw)
		}
	}
	return nil
}

// extractPath returns the path portion of a CLI argument, or "" when the
// argument doesn't look like a filesystem path.
func extractPath(arg string) string {
	if arg == "" || strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") {
		// Bare flags like "--verbose" are not paths.
		// Flags with values like "--out=foo" fall through.
		if strings.HasPrefix(arg, "-") {
			return ""
		}
	}
	if i := strings.Index(arg, "="); i >= 0 {
		arg = arg[i+1:]
	}
	if arg == "" {
		return ""
	}
	if !looksLikePath(arg) {
		return ""
	}
	return arg
}

func looksLikePath(s string) bool {
	if strings.ContainsAny(s, "/\\") {
		return true
	}
	if filepath.IsAbs(s) {
		return true
	}
	if strings.HasPrefix(s, ".") {
		return true
	}
	return false
}

func absPath(p, dir string) (string, error) {
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	base := dir
	if base == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		base = cwd
	}
	return filepath.Clean(filepath.Join(base, p)), nil
}

func insideAny(target string, roots []string) bool {
	for _, r := range roots {
		rel, err := filepath.Rel(r, target)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}

// envWhitelist enumerates the environment variables propagated to children by
// MinimalEnv. Anything else is intentionally dropped to avoid leaking secrets
// (AWS_*, GITHUB_TOKEN, etc.) into untrusted plugins.
var envWhitelist = []string{
	"PATH",
	"HOME",
	"USER",
	"USERNAME",
	"USERPROFILE", // Windows
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
	"TZ",
	"TMPDIR",
	"TEMP",
	"TMP",
	"SystemRoot",  // Windows
	"SystemDrive", // Windows
	"ComSpec",     // Windows
	"PATHEXT",     // Windows
	"WINDIR",      // Windows
	"PROCESSOR_ARCHITECTURE",
	"NUMBER_OF_PROCESSORS",
	"GOROOT",
	"GOPATH",
	"GOCACHE",
	"GOMODCACHE",
	"PYTHONPATH",
	"PYTHONHOME",
	"VIRTUAL_ENV",
	"CARGO_HOME",
	"RUSTUP_HOME",
	// SOURCE_DATE_EPOCH is honored by reproducible-build aware tools
	// (protoc, jar, gzip, …) and Buffalo intentionally forwards it so
	// downstream codegen produces byte-identical output.
	"SOURCE_DATE_EPOCH",
}

// MinimalEnv returns a copy of os.Environ filtered to envWhitelist plus
// BUFFALO_*. The result is safe to extend with ExtraEnv.
func MinimalEnv() []string {
	whitelist := make(map[string]struct{}, len(envWhitelist))
	for _, k := range envWhitelist {
		whitelist[normalizeEnvKey(k)] = struct{}{}
	}
	out := make([]string, 0, 32)
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			continue
		}
		key := normalizeEnvKey(kv[:i])
		if _, ok := whitelist[key]; ok {
			out = append(out, kv)
			continue
		}
		if strings.HasPrefix(strings.ToUpper(kv[:i]), "BUFFALO_") {
			out = append(out, kv)
		}
	}
	return out
}

// normalizeEnvKey accounts for Windows env vars being case-insensitive.
func normalizeEnvKey(k string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(k)
	}
	return k
}

// boundedBuffer is a bytes.Buffer wrapper that stops accepting bytes after a
// configured limit and remembers whether truncation happened.
type boundedBuffer struct {
	buf       bytes.Buffer
	limit     int64
	written   int64
	truncated bool
}

func newBoundedBuffer(limit int64) *boundedBuffer {
	return &boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.limit < 0 {
		return b.buf.Write(p)
	}
	remaining := b.limit - b.written
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil // pretend we wrote, drop bytes
	}
	if int64(len(p)) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.written += remaining
		b.truncated = true
		return len(p), nil
	}
	n, err := b.buf.Write(p)
	b.written += int64(n)
	return n, err
}

func (b *boundedBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

// Compile-time check that we use fmt (kept for future error formatting).
var _ = fmt.Sprintf
