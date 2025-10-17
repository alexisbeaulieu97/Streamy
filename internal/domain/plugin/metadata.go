package plugin

import "fmt"

// Metadata describes plugin identity, version, and dependencies.
type Metadata struct {
	ID           string
	Name         string
	Version      string
	Type         Type
	Description  string
	Dependencies []string
	APIVersion   string
}

// Validate ensures metadata values satisfy invariants.
func (m Metadata) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("plugin id is required")
	}
	if m.Type == "" || !IsSupportedType(m.Type) {
		return fmt.Errorf("unsupported plugin type %q", m.Type)
	}
	if m.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	return nil
}
