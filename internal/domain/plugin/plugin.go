package plugin

// Type represents the category of steps a plugin supports.
type Type string

const (
	TypePackage    Type = "package"
	TypeRepo       Type = "repo"
	TypeSymlink    Type = "symlink"
	TypeCopy       Type = "copy"
	TypeCommand    Type = "command"
	TypeTemplate   Type = "template"
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
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusUnknown  Status = "unknown"
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
