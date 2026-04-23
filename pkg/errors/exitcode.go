package errors

// Exit codes for the buffalo CLI. These follow a stable numbering scheme so
// that scripts and CI pipelines can react to specific failure classes:
//
//	0   success
//	1   generic / unclassified error
//	2   misuse of CLI (bad flags, unknown command) — owned by cobra
//	10  configuration error (missing/invalid buffalo.yaml, bad schema)
//	11  validation error (config or input failed semantic checks)
//	20  IO error (read/write/permission)
//	21  not found (file, target, plugin)
//	30  build / compilation failure
//	31  plugin failure
//	40  dependency / network error
const (
	ExitSuccess    = 0
	ExitError      = 1
	ExitUsage      = 2
	ExitConfig     = 10
	ExitValidation = 11
	ExitIO         = 20
	ExitNotFound   = 21
	ExitBuild      = 30
	ExitPlugin     = 31
	ExitDependency = 40
)

// ExitCode maps an *Error to a stable shell exit code. Returns ExitError for
// anything not matched by a specific category.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	e, ok := err.(*Error)
	if !ok {
		return ExitError
	}
	switch e.Code {
	case ErrConfig, ErrConfigNotFound, ErrConfigInvalid:
		return ExitConfig
	case ErrValidation, ErrInvalidInput, ErrInvalidArgument:
		return ExitValidation
	case ErrNotFound, ErrProtoNotFound, ErrCompilerNotFound:
		return ExitNotFound
	case ErrIO, ErrFileRead, ErrFileWrite, ErrFileDelete:
		return ExitIO
	case ErrCompilation, ErrCompilerVersion:
		return ExitBuild
	case ErrPlugin, ErrPluginLoad, ErrPluginExecute:
		return ExitPlugin
	case ErrDependency, ErrCircularDep, ErrMissingDep:
		return ExitDependency
	}
	return ExitError
}
