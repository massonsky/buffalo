package config

import (
	"strings"

	"github.com/spf13/viper"
)

// EnvPrefix is the prefix all Buffalo environment variables share.
const EnvPrefix = "BUFFALO"

// envBindKeys lists nested viper keys that must be bound to BUFFALO_* env
// vars eagerly (before any YAML is loaded). AutomaticEnv only inspects keys
// that viper already knows about, so nested struct fields require explicit
// binds to be overridable from the environment.
//
// Keep in sync with the Config struct.
var envBindKeys = []string{
	"schema_version",
	"project.name",
	"project.version",
	"output.base_dir",
	"build.workers",
	"build.incremental",
	"build.cache.enabled",
	"build.cache.directory",
	"logging.level",
	"logging.format",
	"logging.output",
	"logging.file",

	"languages.go.enabled",
	"languages.go.module",
	"languages.go.generator",
	"languages.go.models_output",
	"languages.go.orm",
	"languages.go.orm_plugin",

	"languages.python.enabled",
	"languages.python.package",
	"languages.python.generator",
	"languages.python.workdir",
	"languages.python.models_output",
	"languages.python.orm",
	"languages.python.orm_plugin",
	"languages.python.pb2_import_prefix",

	"languages.rust.enabled",
	"languages.rust.generator",
	"languages.rust.models_output",
	"languages.rust.orm",
	"languages.rust.orm_plugin",

	"languages.cpp.enabled",
	"languages.cpp.namespace",
	"languages.cpp.models_output",
	"languages.cpp.orm",
	"languages.cpp.orm_plugin",

	"languages.typescript.enabled",
	"languages.typescript.generator",
	"languages.typescript.output",
	"languages.typescript.models_output",
	"languages.typescript.orm",
	"languages.typescript.orm_plugin",

	"bazel.enabled",
	"bazel.bazel_path",
	"bazel.auto_detect",
	"bazel.go_module_path",
	"bazel.strip_import_prefix",
	"bazel.use_query",
	"bazel.generate_build_files",

	"models.enabled",
	"models.generate_models_from_proto",
}

// ApplyEnv configures the supplied viper instance to honor BUFFALO_* env
// overrides for every nested key declared in envBindKeys. Safe to call on
// both the global viper (`viper.GetViper()`) and on instances created via
// `viper.New()` from LoadFromFile.
func ApplyEnv(v *viper.Viper) {
	if v == nil {
		return
	}
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	for _, k := range envBindKeys {
		_ = v.BindEnv(k)
	}
}

// EnvBindKeys returns a copy of the registered env-bound keys for tests and
// docs.
func EnvBindKeys() []string {
	out := make([]string, len(envBindKeys))
	copy(out, envBindKeys)
	return out
}
