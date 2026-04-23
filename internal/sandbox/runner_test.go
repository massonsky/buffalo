package sandbox

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRun_EchoSuccess(t *testing.T) {
	name, args := echoCmd("hello")
	res, err := Run(context.Background(), Options{Name: name, Args: args})
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, res.Stderr)
	}
	if !strings.Contains(string(res.Stdout), "hello") {
		t.Fatalf("stdout missing 'hello': %q", res.Stdout)
	}
}

func TestRun_TimeoutKills(t *testing.T) {
	name, args := sleepCmd(5)
	start := time.Now()
	_, err := Run(context.Background(), Options{
		Name:      name,
		Args:      args,
		Timeout:   200 * time.Millisecond,
		WaitDelay: 200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("timeout did not kill child quickly: %s", time.Since(start))
	}
}

func TestRun_OutputBounded(t *testing.T) {
	name, args := bigOutputCmd(2048)
	res, err := Run(context.Background(), Options{
		Name:           name,
		Args:           args,
		MaxOutputBytes: 256,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.StdoutTrunc {
		t.Fatalf("expected stdout to be truncated, got %d bytes", len(res.Stdout))
	}
	if int64(len(res.Stdout)) > 256 {
		t.Fatalf("stdout exceeded limit: %d", len(res.Stdout))
	}
}

func TestRun_AllowedRootsRejectsEscape(t *testing.T) {
	name, args := echoCmd("ok")
	args = append(args, "../../etc/passwd")
	_, err := Run(context.Background(), Options{
		Name:         name,
		Args:         args,
		Dir:          t.TempDir(),
		AllowedRoots: []string{"."},
	})
	if err == nil {
		t.Fatal("expected validation error for escaping path")
	}
	if !strings.Contains(err.Error(), "outside allowed roots") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_AllowedRootsAcceptsContained(t *testing.T) {
	dir := t.TempDir()
	name, args := echoCmd("ok")
	args = append(args, "./inner/file.proto")
	_, err := Run(context.Background(), Options{
		Name:         name,
		Args:         args,
		Dir:          dir,
		AllowedRoots: []string{dir},
	})
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestMinimalEnv_DropsSecrets(t *testing.T) {
	t.Setenv("AWS_SECRET_ACCESS_KEY", "shh")
	t.Setenv("BUFFALO_LOG_LEVEL", "debug")
	t.Setenv("PATH", "/usr/bin")

	env := MinimalEnv()
	for _, kv := range env {
		if strings.HasPrefix(kv, "AWS_SECRET_ACCESS_KEY=") {
			t.Fatalf("MinimalEnv leaked secret: %q", kv)
		}
	}
	var sawBuffalo, sawPath bool
	for _, kv := range env {
		if strings.HasPrefix(kv, "BUFFALO_LOG_LEVEL=") {
			sawBuffalo = true
		}
		if strings.HasPrefix(kv, "PATH=") {
			sawPath = true
		}
	}
	if !sawBuffalo {
		t.Error("BUFFALO_* not propagated")
	}
	if !sawPath {
		t.Error("PATH not propagated")
	}
}

// --- platform helpers ---

func echoCmd(s string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", "echo", s}
	}
	return "/bin/sh", []string{"-c", "echo " + s}
}

func sleepCmd(seconds int) (string, []string) {
	if runtime.GOOS == "windows" {
		// ping -n N+1 sleeps roughly N seconds without needing a sleep utility.
		return "cmd", []string{"/c", "ping", "-n", itoa(seconds + 1), "127.0.0.1", ">", "NUL"}
	}
	return "/bin/sh", []string{"-c", "sleep " + itoa(seconds)}
}

func bigOutputCmd(n int) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", "for /L %i in (1,1," + itoa(n) + ") do @echo x"}
	}
	return "/bin/sh", []string{"-c", "yes x | head -n " + itoa(n)}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
