package plugin

import (
	"fmt"
	"sync"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Plugin)
)

// RegisterPlugin adds a plugin implementation for the provided type.
func RegisterPlugin(stepType string, p Plugin) error {
	if p == nil {
		return streamyerrors.NewPluginError(stepType, fmt.Errorf("plugin is nil"))
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[stepType]; exists {
		return streamyerrors.NewPluginError(stepType, fmt.Errorf("plugin already registered"))
	}

	registry[stepType] = p
	return nil
}

// GetPlugin retrieves a plugin by type.
func GetPlugin(stepType string) (Plugin, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	plugin, ok := registry[stepType]
	if !ok {
		return nil, streamyerrors.NewPluginError(stepType, fmt.Errorf("no plugin registered"))
	}

	return plugin, nil
}

// ResetRegistry clears plugin registrations (for tests).
func ResetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]Plugin)
}
