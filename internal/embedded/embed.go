// Package embedded provides proto files bundled into the Buffalo binary.
//
// When Buffalo is installed via `go install`, the proto files for
// buffalo.validate are embedded into the binary and can be extracted
// to the user's project workspace at any time.
//
// This solves the problem of proto files not being available after
// `go install`, since Go only installs the compiled binary.
package embedded

import "embed"

// ProtoFS contains all embedded proto files from the proto/ directory tree.
// The files are available at runtime and can be extracted to disk via
// ExtractValidateProto().
//
//go:embed proto
var ProtoFS embed.FS
