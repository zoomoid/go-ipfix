/*
Copyright 2023 Alexander Bartolomey (github@alexanderbartolomey.de)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ipfix

import (
	"context"
	"encoding/json"
	"sync"
)

// EphemeralCache is the most basic of in-memory caches. It is memory-safe
// by using a Read-Write mutex on all accessing functions.
// It does not expire entries automatically and does not persist anything on
// disk, nor does it support recovery
type EphemeralCache struct {
	templates map[TemplateKey]*Template

	mu *sync.RWMutex

	name string
}

var _ TemplateCache = &EphemeralCache{}

// NewBasicTemplateCache creates a new in-memory template cache that lives for the lifetime
// of the caller
func NewDefaultEphemeralCache() StatefulTemplateCache {
	return NewNamedEphemeralCache("default")
}

func NewNamedEphemeralCache(name string) StatefulTemplateCache {
	ts := &EphemeralCache{
		templates: make(map[TemplateKey]*Template),
		mu:        &sync.RWMutex{},
		name:      name,
	}
	return ts
}

func (ts *EphemeralCache) GetAll(ctx context.Context) map[TemplateKey]*Template {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	t := ts.templates
	return t
}

func (ts *EphemeralCache) Get(ctx context.Context, key TemplateKey) (*Template, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	template, ok := ts.templates[key]
	if !ok {
		return nil, templateNotFound(key.ObservationDomainId, key.TemplateId)
	}
	return template, nil
}

func (ts *EphemeralCache) Delete(ctx context.Context, key TemplateKey) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	delete(ts.templates, key)
	return nil
}

func (ts *EphemeralCache) Add(ctx context.Context, key TemplateKey, template *Template) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.templates[key] = template

	return nil
}

func (ts *EphemeralCache) Type() string {
	return "ephemeral"
}

func (ts *EphemeralCache) Name() string {
	return ts.name
}

func (ts *EphemeralCache) MarshalJSON() ([]byte, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	s := make(map[string]interface{})
	for k, v := range ts.templates {
		s[k.String()] = v
	}
	return json.Marshal(s)
}

func (ts *EphemeralCache) Close(context.Context) error {
	// no-op
	return nil
}

func (ts *EphemeralCache) Initialize(context.Context) error {
	// no-op
	return nil
}

func (ts *EphemeralCache) Prepare() error {
	// no-op
	return nil
}

func (ts *EphemeralCache) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
