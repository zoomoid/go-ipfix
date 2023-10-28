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
	"fmt"
	"sync"
	"time"
)

type templateElement struct {
	deadline time.Time
	created  time.Time

	expired bool

	template *Template
}

type DecayingEphemeralCache struct {
	templates map[TemplateKey]templateElement

	timeout time.Duration

	mu *sync.RWMutex

	name string
}

var _ TemplateCacheWithTimeout = &DecayingEphemeralCache{}

func NewDefaultDecayingEphemeralCache() TemplateCache {
	return NewNamedDecayingEphemeralCache("default")
}

func NewNamedDecayingEphemeralCache(name string) TemplateCache {
	return &DecayingEphemeralCache{
		templates: make(map[TemplateKey]templateElement),
		mu:        &sync.RWMutex{},
		name:      name,
		timeout:   0,
	}
}

func (ts *DecayingEphemeralCache) GetAll(ctx context.Context) map[TemplateKey]*Template {
	ts.expireTemplates()

	ts.mu.RLock()
	defer ts.mu.RUnlock()

	mm := make(map[TemplateKey]*Template, len(ts.templates))
	for k, v := range ts.templates {
		mm[k] = v.template
	}
	return mm
}

func (ts *DecayingEphemeralCache) Get(ctx context.Context, key TemplateKey) (*Template, error) {
	ts.expireTemplates()

	ts.mu.RLock()
	defer ts.mu.RUnlock()

	te, ok := ts.templates[key]
	if !ok {
		return nil, TemplateNotFound(key.ObservationDomainId, key.TemplateId)
	}

	if te.expired {
		return nil, fmt.Errorf("template %d expired for domain %d", key.TemplateId, key.ObservationDomainId)
	}

	return te.template, nil
}

func (ts *DecayingEphemeralCache) Add(ctx context.Context, key TemplateKey, template *Template) error {
	ts.expireTemplates()

	ts.mu.Lock()
	defer ts.mu.Unlock()

	created := time.Now()
	deadline := created.Add(ts.timeout)

	ts.templates[key] = templateElement{
		created:  created,
		deadline: deadline,
		expired:  false,
		template: template,
	}
	return nil
}

func (t *DecayingEphemeralCache) Delete(ctx context.Context, key TemplateKey) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.templates, key)
	return nil
}

// SetTimeout updates the internal duration stored for calculating deadlines on addition of (new) templates.
// NOTE that SetTimeout does not alter any existing deadlines, i.e., if the timeout duration is changed during
// runtime, and there are already templates in the cache, a longer or shorter timeout duration does NOT affect
// the deadline of those templates, only new ones
func (ts *DecayingEphemeralCache) SetTimeout(d time.Duration) {
	ts.expireTemplates()

	ts.timeout = d
}

func (ts *DecayingEphemeralCache) Type() string {
	return "decaying_ephemeral"
}

func (ts *DecayingEphemeralCache) Name() string {
	return ts.name
}

func (ts *DecayingEphemeralCache) MarshalJSON() ([]byte, error) {
	ts.expireTemplates()

	ts.mu.RLock()
	defer ts.mu.RUnlock()

	s := make(map[string]interface{})
	for k, v := range ts.templates {
		if !v.expired {
			s[k.String()] = v
		}
	}
	return json.Marshal(s)
}

func (ts *DecayingEphemeralCache) expireTemplates() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	for _, v := range ts.templates {
		if time.Now().After(v.deadline) {
			// template has surpassed its deadline, mark it as expired. Subsequent access
			// to the template via Get() will return an error saying the template expired.
			// This is done to differentiate between expiry and non-existence
			v.expired = true
		}
	}
}

func (ts *DecayingEphemeralCache) Close(context.Context) error {
	// no-op
	return nil
}

func (ts *DecayingEphemeralCache) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
