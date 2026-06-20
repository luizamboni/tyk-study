package token

import (
	"sync"
	"time"
)

const expirationMargin = 5 * time.Second

type Value struct {
	AccessToken string
	ExpiresAt   time.Time
}

type Result struct {
	Value        Value
	PluginSource string
	BrokerSource string
}

type pendingResolution struct {
	done   chan struct{}
	result Result
	err    error
}

type Resolver struct {
	mutex   sync.Mutex
	cache   map[string]Value
	pending map[string]*pendingResolution
	broker  *Broker
}

func NewResolver(broker *Broker) *Resolver {
	return &Resolver{
		cache:   make(map[string]Value),
		pending: make(map[string]*pendingResolution),
		broker:  broker,
	}
}

func (resolver *Resolver) Resolve(tenant, service string) (Result, error) {
	key := tenant + ":" + service

	resolver.mutex.Lock()
	if value, found := resolver.validCachedValue(key); found {
		resolver.mutex.Unlock()
		return Result{
			Value:        value,
			PluginSource: "plugin-cache",
			BrokerSource: "not-called",
		}, nil
	}
	if pending, found := resolver.pending[key]; found {
		resolver.mutex.Unlock()
		<-pending.done
		return pending.result, pending.err
	}
	pending := &pendingResolution{done: make(chan struct{})}
	resolver.pending[key] = pending
	resolver.mutex.Unlock()

	value, brokerSource, err := resolver.broker.Resolve(tenant, service)
	result := Result{
		Value:        value,
		PluginSource: "token-broker",
		BrokerSource: brokerSource,
	}
	resolver.finishResolution(key, pending, result, err)

	return result, err
}

func (resolver *Resolver) validCachedValue(key string) (Value, bool) {
	value, found := resolver.cache[key]
	if !found || time.Now().After(value.ExpiresAt.Add(-expirationMargin)) {
		return Value{}, false
	}

	return value, true
}

func (resolver *Resolver) finishResolution(
	key string,
	pending *pendingResolution,
	result Result,
	err error,
) {
	resolver.mutex.Lock()
	defer resolver.mutex.Unlock()

	pending.result = result
	pending.err = err
	if err == nil {
		resolver.cache[key] = result.Value
	}
	delete(resolver.pending, key)
	close(pending.done)
}
