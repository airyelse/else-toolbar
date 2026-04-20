package runtime

import "sync"

// SDKRegistry holds metadata and logic for a single SDK type.
type SDKRegistry struct {
	Name     string          // Display name, e.g. "Go"
	Icon     string          // Theme color or icon identifier
	Provider SDKProvider     // Download / checksum / post-extract
	Fetcher  func() []string // Fetch available versions list
}

var (
	registryMu    sync.RWMutex
	registry      = make(map[SDKType]*SDKRegistry)
	registryOrder []SDKType
)

// RegisterSDK adds an SDK to the registry. Call this in init().
func RegisterSDK(sdkType SDKType, reg *SDKRegistry) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[sdkType]; !exists {
		registryOrder = append(registryOrder, sdkType)
	}
	registry[sdkType] = reg
}

// RegisteredSDKs returns all registered SDK types in registration order.
func RegisteredSDKs() []SDKType {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make([]SDKType, len(registryOrder))
	copy(result, registryOrder)
	return result
}

// GetRegistry returns the registry entry for a given SDK type, or nil.
func GetRegistry(sdkType SDKType) *SDKRegistry {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[sdkType]
}
