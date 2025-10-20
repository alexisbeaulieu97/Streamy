package plugin

// Type represents the category of steps a plugin supports.
type Type string

const (
	// TypePackage identifies a package management plugin.
	TypePackage Type = "package"
	// TypeRepo identifies a repository management plugin.
	TypeRepo Type = "repo"
	// TypeSymlink identifies a symbolic link management plugin.
	TypeSymlink Type = "symlink"
	// TypeCopy identifies a file copy plugin.
	TypeCopy Type = "copy"
	// TypeCommand identifies a command execution plugin.
	TypeCommand Type = "command"
	// TypeTemplate identifies a templating plugin.
	TypeTemplate Type = "template"
	// TypeLineInFile identifies a line-in-file plugin.
	TypeLineInFile Type = "line_in_file"
)

var supportedTypes = []Type{
	TypePackage,
	TypeRepo,
	TypeSymlink,
	TypeCopy,
	TypeCommand,
	TypeTemplate,
	TypeLineInFile,
}

// Status captures the lifecycle state for a plugin.
type Status string

const (
	// StatusActive indicates the plugin is enabled.
	StatusActive Status = "active"
	// StatusDisabled indicates the plugin is temporarily disabled.
	StatusDisabled Status = "disabled"
	// StatusUnknown indicates the plugin state could not be determined.
	StatusUnknown Status = "unknown"
)

// Plugin defines the contract that domain services expect from plugin implementations.
type Plugin interface {
	Metadata() Metadata
}

// IsSupportedType reports whether the provided type is recognised.
func IsSupportedType(t Type) bool {
	for _, candidate := range supportedTypes {
		if candidate == t {
			return true
		}
	}

	return false
}
