package plugin

// DefaultManager is the global plugin manager instance
var DefaultManager *Manager

func init() {
	// Initialize the default manager
	DefaultManager = NewManager()
}

// GetManager returns the default plugin manager
func GetManager() *Manager {
	return DefaultManager
}
