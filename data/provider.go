package data

// Provider can provide a state or store database upon request. Either can be
// nil even if requested.
type Provider interface {
	State(network string) *State
	Store() *Store
}
