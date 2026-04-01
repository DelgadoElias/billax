package provider

import (
	"fmt"
	"sync"

	"github.com/DelgadoElias/billax/internal/errors"
)

// Registry holds all registered payment providers keyed by name
type Registry struct {
	mu        sync.RWMutex
	providers map[string]PaymentProvider
}

// NewRegistry creates a new empty provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]PaymentProvider),
	}
}

// Register adds a payment provider to the registry
// Panics if a provider with the same name is already registered
func (r *Registry) Register(provider PaymentProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.GetProviderName()
	if _, exists := r.providers[name]; exists {
		panic(fmt.Sprintf("provider %q is already registered", name))
	}

	r.providers[name] = provider
}

// Lookup retrieves a payment provider by name
// Returns ErrProviderNotFound if the provider is not registered
func (r *Registry) Lookup(name string) (PaymentProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %q: %w", name, errors.ErrProviderNotFound)
	}

	return provider, nil
}

// Names returns a list of all registered provider names
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}
