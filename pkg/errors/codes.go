package errors

// ErrorCode represents a category of errors.
type ErrorCode string

// Common error codes used throughout Buffalo.
const (
	// ErrUnknown represents an unknown error.
	ErrUnknown ErrorCode = "UNKNOWN"

	// ErrInternal represents an internal error.
	ErrInternal ErrorCode = "INTERNAL"

	// ErrInvalidInput represents invalid input.
	ErrInvalidInput ErrorCode = "INVALID_INPUT"

	// ErrInvalidArgument represents invalid argument passed to a function.
	ErrInvalidArgument ErrorCode = "INVALID_ARGUMENT"

	// ErrNotFound represents a resource not found error.
	ErrNotFound ErrorCode = "NOT_FOUND"

	// ErrAlreadyExists represents a resource already exists error.
	ErrAlreadyExists ErrorCode = "ALREADY_EXISTS"

	// ErrPermissionDenied represents a permission denied error.
	ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"

	// ErrTimeout represents a timeout error.
	ErrTimeout ErrorCode = "TIMEOUT"

	// ErrCanceled represents a canceled operation.
	ErrCanceled ErrorCode = "CANCELED"

	// Configuration errors
	ErrConfig         ErrorCode = "CONFIG_ERROR"
	ErrConfigNotFound ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigInvalid  ErrorCode = "CONFIG_INVALID"

	// Proto file errors
	ErrProtoNotFound ErrorCode = "PROTO_NOT_FOUND"
	ErrProtoInvalid  ErrorCode = "PROTO_INVALID"
	ErrProtoScan     ErrorCode = "PROTO_SCAN_ERROR"

	// Compilation errors
	ErrCompilation      ErrorCode = "COMPILATION_FAILED"
	ErrCompilerNotFound ErrorCode = "COMPILER_NOT_FOUND"
	ErrCompilerVersion  ErrorCode = "COMPILER_VERSION_ERROR"

	// Dependency errors
	ErrDependency  ErrorCode = "DEPENDENCY_ERROR"
	ErrCircularDep ErrorCode = "CIRCULAR_DEPENDENCY"
	ErrMissingDep  ErrorCode = "MISSING_DEPENDENCY"

	// Validation errors
	ErrValidation ErrorCode = "VALIDATION_ERROR"

	// IO errors
	ErrIO         ErrorCode = "IO_ERROR"
	ErrFileRead   ErrorCode = "FILE_READ_ERROR"
	ErrFileWrite  ErrorCode = "FILE_WRITE_ERROR"
	ErrFileDelete ErrorCode = "FILE_DELETE_ERROR"

	// Cache errors
	ErrCache      ErrorCode = "CACHE_ERROR"
	ErrCacheRead  ErrorCode = "CACHE_READ_ERROR"
	ErrCacheWrite ErrorCode = "CACHE_WRITE_ERROR"

	// Plugin errors
	ErrPlugin        ErrorCode = "PLUGIN_ERROR"
	ErrPluginLoad    ErrorCode = "PLUGIN_LOAD_ERROR"
	ErrPluginExecute ErrorCode = "PLUGIN_EXECUTE_ERROR"
)

// String returns the string representation of the error code.
func (c ErrorCode) String() string {
	return string(c)
}
