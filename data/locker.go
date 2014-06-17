package data

// Locker is an interface that allows locking and unlocking of the databases.
type Locker interface {
	// ReadingState calls a callback if the state is enabled.
	// The returned boolean is whether or not the function was called.
	ReadState(networkID string, fn func(*State)) bool
	// OpenState locks the state db and returns it.
	OpenState(networkID string) *State
	// CloseState unlocks the state db after use by OpenState.
	CloseState(networkID string)

	// ReadStore calls a callback if the store is enabled.
	// The returned boolean is whether or not the function was called.
	ReadStore(fn func(*Store)) bool
	// OpenStore locks the store db, and returns it
	OpenReadStore() *Store
	// CloseStore unlocks the store db after use by OpenState.
	CloseReadStore()

	// WriteStore calls a callback if the store is enabled.
	// The returned boolean is whether or not the function was called.
	WriteStore(fn func(*Store)) bool
	// OpenWriteStore locks the store db, and returns it
	OpenWriteStore() *Store
	// CloseWriteStore unlocks the store db after use by OpenWriteStore.
	CloseWriteStore()
}
