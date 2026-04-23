package builder

import "github.com/massonsky/buffalo/internal/config"

// pinnedToolVersions extracts (name, version) pairs from cfg.Tools so they
// can be folded into the cache key. Returns nil when nothing is pinned so
// pre-existing cache entries (without ToolsHash) remain valid until the user
// opts in.
func pinnedToolVersions(cfg *config.Config) map[string]string {
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
