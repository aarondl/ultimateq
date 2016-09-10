package registrar

import "sync"

// Proxy all the registrations through a storage mechanism, the important
// thing about proxy is that it adds a "name" layer to each registration
// for say extensions or the like. It also has a provision for unregistering
// any of the things by its proxied by name.
type Proxy struct {
	registrar Interface

	mut     sync.Mutex
	holders map[string]*holder
}

// NewProxy constructor, holds a reference to the interface passed in.
func NewProxy(registrar Interface) *Proxy {
	p := &Proxy{
		registrar: registrar,
		holders:   make(map[string]*holder),
	}

	return p
}

// Get a proxying object and a kill channel for it. Creates a new one if
// one is not found.
func (p *Proxy) Get(name string) Interface {
	p.mut.Lock()
	defer p.mut.Unlock()

	holder, ok := p.holders[name]
	if ok {
		return holder
	}

	holder = newHolder(p.registrar)
	p.holders[name] = holder
	return holder
}

// Unregister everything registered to name
func (p *Proxy) Unregister(name string) {
	p.mut.Lock()
	defer p.mut.Unlock()

	holder, ok := p.holders[name]
	if !ok {
		return
	}
	delete(p.holders, name)

	holder.unregisterAll()
}
