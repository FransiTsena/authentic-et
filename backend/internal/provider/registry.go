package provider

import (
	"fmt"
	"strings"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]PaymentVerifierProvider
}

func NewRegistry() *Registry {
	r := &Registry{
		providers: make(map[string]PaymentVerifierProvider),
	}

	// Register default providers
	r.Register(NewCBEProvider())
	r.Register(NewTelebirrProvider())
	r.Register(NewBOAProvider())
	r.Register(NewDashenProvider())
	r.Register(NewAwashProvider())
	r.Register(NewMpesaProvider())
	r.Register(NewMockProvider())

	return r
}

func (r *Registry) Register(p PaymentVerifierProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(p.ProviderKey()))
	r.providers[key] = p
}

func (r *Registry) Get(providerKey string) (PaymentVerifierProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(providerKey))
	p, ok := r.providers[key]
	if !ok {
		return nil, fmt.Errorf("provider '%s' is not supported", providerKey)
	}
	return p, nil
}

type ProviderInfo struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
}

func (r *Registry) List() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, ProviderInfo{
			Key:         p.ProviderKey(),
			DisplayName: p.DisplayName(),
		})
	}
	return list
}
