package data

// Locker is an interface that allows locking and unlocking of the databases.
type Locker interface {
	// UsingState calls a callback if the state is enabled.
	// The returned boolean is whether or not the function was called.
	UsingState(networkID string, fn func(*State)) bool
	// OpenState locks the state db and returns it.
	OpenState(networkID string) *State
	// CloseState unlocks the state db after use by OpenState.
	CloseState(networkID string)

	// UsingStore calls a callback if the store is enabled.
	// The returned boolean is whether or not the function was called.
	UsingStore(fn func(*Store)) bool
	// OpenStore locks the store db, and returns it
	OpenStore() *Store
	// CloseStore unlocks the store db after use by OpenState.
	CloseStore()
}
