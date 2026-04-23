package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/pkg/errors"
)

// toolsPinVerifyCmd is registered as `buffalo tools verify`. It is separate
// from the legacy environment-checking `tools check` command: this one
// validates the explicit pins recorded in buffalo.yaml under `tools:` and
// is intended for CI gates.
var toolsPinVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify pinned protoc / protoc-gen-* binaries (version + sha256)",
	Long: `For every entry in tools.protoc and tools.plugins:
  * locates the binary (config 'path' override, then PATH);
  * runs '<bin> --version' and ensures the configured Version is a substring
    of the output;
  * when 'sha256' is set, recomputes the digest of the binary and compares.

Exit code is non-zero when any pin is missing or mismatched, making this
suitable for CI gates and 'buffalo build --frozen-lockfile' preflight.`,
	RunE: runToolsPinVerify,
}

func init() {
	toolsCmd.AddCommand(toolsPinVerifyCmd)
}

func runToolsPinVerify(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	cfg, err := loadConfig(log)
	if err != nil {
		return err
	}

	pins := collectToolPins(cfg)
	if len(pins) == 0 {
		log.Info("No tools pinned. Add a 'tools:' section to buffalo.yaml.")
		return nil
	}

	var failures []string
	for _, p := range pins {
		bin, lookupErr := resolvePinnedToolPath(p.binaryName, p.pin.Path)
		if lookupErr != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", p.label, lookupErr))
			log.Error("❌ " + p.label + ": " + lookupErr.Error())
			continue
		}

		if err := verifyToolVersion(bin, p.pin.Version); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", p.label, err))
			log.Error("❌ " + p.label + ": " + err.Error())
			continue
		}

		if p.pin.Sha256 != "" {
			if err := verifyToolSha256(bin, p.pin.Sha256); err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", p.label, err))
				log.Error("❌ " + p.label + ": " + err.Error())
				continue
			}
		}

		log.Info("✅ " + p.label + " " + p.pin.Version + " @ " + bin)
	}

	if len(failures) > 0 {
		return errors.New(errors.ErrConfig, fmt.Sprintf("%d tool pin(s) failed verification", len(failures)))
	}
	return nil
}

type toolPinEntry struct {
	label      string
	binaryName string
	pin        config.ToolPin
}

func collectToolPins(cfg *config.Config) []toolPinEntry {
	if cfg == nil {
		return nil
	}
	out := make([]toolPinEntry, 0, 1+len(cfg.Tools.Plugins))
	if cfg.Tools.Protoc != nil && cfg.Tools.Protoc.Version != "" {
		out = append(out, toolPinEntry{
			label:      "protoc",
			binaryName: "protoc",
			pin:        *cfg.Tools.Protoc,
		})
	}
	keys := make([]string, 0, len(cfg.Tools.Plugins))
	for k := range cfg.Tools.Plugins {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, toolPinEntry{
			label:      "protoc-gen-" + k,
			binaryName: "protoc-gen-" + k,
			pin:        cfg.Tools.Plugins[k],
		})
	}
	return out
}

// resolvePinnedToolPath honors an explicit override path, otherwise consults PATH.
func resolvePinnedToolPath(name, override string) (string, error) {
	if override != "" {
		if _, err := os.Stat(override); err != nil {
			return "", errors.Wrap(err, errors.ErrNotFound, "configured path is not accessible")
		}
		return override, nil
	}
	bin, err := exec.LookPath(name)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrNotFound, "binary not found on PATH")
	}
	return bin, nil
}

// verifyToolVersion runs '<bin> --version' and checks for substring match.
func verifyToolVersion(bin, want string) error {
	if want == "" {
		return nil
	}
	cmd := exec.Command(bin, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, errors.ErrInternal, "failed to invoke --version")
	}
	got := strings.TrimSpace(string(out))
	if !strings.Contains(got, want) {
		return errors.New(errors.ErrConfig, fmt.Sprintf("version mismatch: want %q, got %q", want, got))
	}
	return nil
}

// verifyToolSha256 hashes the binary at path and compares with want (hex).
func verifyToolSha256(path, want string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to read binary for hashing")
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return errors.New(errors.ErrConfig, fmt.Sprintf("sha256 mismatch: want %s, got %s", want, got))
	}
	return nil
}

// PinnedToolVersions returns the (name, version) map suitable for mixing into
// the build-cache key. Returns nil when no tools are pinned so existing cache
// entries remain valid until the user opts in.
func PinnedToolVersions(cfg *config.Config) map[string]string {
	if cfg == nil {
		return nil
	}
	out := make(map[string]string)
	if cfg.Tools.Protoc != nil && cfg.Tools.Protoc.Version != "" {
		out["protoc"] = cfg.Tools.Protoc.Version
	}
	for k, v := range cfg.Tools.Plugins {
		if v.Version != "" {
			out["protoc-gen-"+k] = v.Version
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
