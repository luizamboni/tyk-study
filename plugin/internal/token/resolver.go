package token

import (
	"sync"
	"time"
)

const expirationMargin = 5 * time.Second

type cachedEntry struct {
	cred     Credential
	expiresAt time.Time
}

type pendingResolution struct {
	done chan struct{}
	cred Credential
	err  error
}

type Resolver struct {
	mutex   sync.Mutex
	cache   map[string]cachedEntry
	pending map[string]*pendingResolution
	broker  *Broker
}

func NewResolver(broker *Broker) *Resolver {
	return &Resolver{
		cache:   make(map[string]cachedEntry),
		pending: make(map[string]*pendingResolution),
		broker:  broker,
	}
}

func (r *Resolver) Resolve(tenant, integration, operation, method string) (Credential, error) {
	key := tenant + ":" + integration + ":" + operation

	r.mutex.Lock()
	if entry, found := r.validCached(key); found {
		r.mutex.Unlock()
		return entry.cred, nil
	}
	if p, found := r.pending[key]; found {
		r.mutex.Unlock()
		<-p.done
		return p.cred, p.err
	}
	p := &pendingResolution{done: make(chan struct{})}
	r.pending[key] = p
	r.mutex.Unlock()

	cred, err := r.broker.Resolve(tenant, integration, operation, method)

	r.mutex.Lock()
	p.cred = cred
	p.err = err
	if err == nil && !cred.ExpiresAt.IsZero() {
		r.cache[key] = cachedEntry{cred: cred, expiresAt: cred.ExpiresAt}
	}
	delete(r.pending, key)
	close(p.done)
	r.mutex.Unlock()

	return cred, err
}

func (r *Resolver) validCached(key string) (cachedEntry, bool) {
	entry, found := r.cache[key]
	if !found || time.Now().After(entry.expiresAt.Add(-expirationMargin)) {
		return cachedEntry{}, false
	}
	return entry, true
}
