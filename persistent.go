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
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// PersistentCache uses an InMemoryStore, but can restore and dump its contents to a
// file given to the cache at startup

type PersistentCache struct {
	file *os.File

	// fieldCache is required for injecting into TemplateRecords and
	// subsequently Fields during reconstruction from JSON
	fieldCache FieldCache

	// cache is required for injecting into TemplateRecords and
	// subsequently Fields during reconstruction from JSON
	cache StatefulTemplateCache

	mu *sync.RWMutex

	// wg *sync.WaitGroup

	name string

	ready bool
}

var _ StatefulTemplateCache = &PersistentCache{}
var _ TemplateCacheDriver = &PersistentCache{}

func NewDefaultPersistentCache(file *os.File, fieldCache FieldCache, templateCache StatefulTemplateCache) StatefulTemplateCache {
	return NewNamedPersistentCache("default", file, fieldCache, templateCache)
}

func NewNamedPersistentCache(name string, file *os.File, fieldCache FieldCache, templateCache StatefulTemplateCache) StatefulTemplateCache {
	c := &PersistentCache{
		file:       file,
		fieldCache: fieldCache,
		cache:      templateCache,
		mu:         &sync.RWMutex{},
		// wg:         &sync.WaitGroup{},
		name:  name,
		ready: false,
	}

	// immediately lock mutex to prevent frontend functions from passing by Start/Initialize
	c.mu.Lock()

	return c
}

func (t *PersistentCache) Add(ctx context.Context, key TemplateKey, template *Template) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.cache.Add(ctx, key, template)
}

func (t *PersistentCache) Delete(ctx context.Context, key TemplateKey) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.cache.Delete(ctx, key)
}

func (t *PersistentCache) Get(ctx context.Context, key TemplateKey) (*Template, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.cache.Get(ctx, key)
}

func (t *PersistentCache) GetAll(ctx context.Context) map[TemplateKey]*Template {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.cache.GetAll(ctx)
}

func (t *PersistentCache) MarshalJSON() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	type its struct {
		Type  string          `json:"type,omitempty"`
		Name  string          `json:"name,omitempty"`
		Cache json.RawMessage `json:"cache,omitempty"`
	}

	cc, err := t.cache.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return json.Marshal(its{
		Type:  t.Type(),
		Name:  t.Name(),
		Cache: cc,
	})
}

func (t *PersistentCache) Name() string {
	return t.name
}

func (t *PersistentCache) Type() string {
	return fmt.Sprintf("%s/%s", "persistent", t.cache.Type())
}

func (t *PersistentCache) Prepare() error {
	// check if file is ready etc.
	return nil
}

func (t *PersistentCache) Initialize(ctx context.Context) error {
	// restore templates from JSON
	b, err := io.ReadAll(t.file)
	if err != nil {
		return err
	}

	type marshalledTemplates struct {
		ExportedAt time.Time                  `json:"exported_at,omitempty"`
		StoreType  string                     `json:"store_type,omitempty"`
		StoreName  string                     `json:"store_name,omitempty"`
		Templates  map[string]json.RawMessage `json:"templates,omitempty"`
	}

	ts := marshalledTemplates{}
	err = json.Unmarshal(b, &ts)
	if err != nil {
		return err
	}
	// logger.V(1).Info("restoring templates from file", "store_name", ts.StoreName, "store_type", ts.StoreType, "exported_at", ts.ExportedAt)

	templateMap := make(map[TemplateKey]Template)
	for key, value := range ts.Templates {

		tt := Template{}
		err := json.Unmarshal(value, &tt)
		if err != nil {
			return err
		}

		kkey := TemplateKey{}
		err = kkey.UnmarshalText([]byte(key))
		if err != nil {
			return err
		}

		templateMap[kkey] = tt
	}

	for k, v := range templateMap {
		// pass through mutex/waitgroup of PersistentCache's Add
		err := t.cache.Add(ctx, k, &v)
		if err != nil {
			return err
		}
	}

	// logger.V(1).Info("restored templates from file", "number_of_templates", len(templateMap))

	return nil
}

func (t *PersistentCache) Close(context.Context) error {
	fn := t.file.Name()

	// close file for reading access
	err := t.file.Close()
	if err != nil {
		return err
	}

	// re-open file for writing access
	file, err := os.Create(fn)
	if err != nil {
		return err
	}
	t.file = file
	defer t.file.Close()

	// dump templates to JSON, write to file and close handle
	type templates struct {
		ExportedAt time.Time       `json:"exported_at,omitempty"`
		StoreType  string          `json:"store_type,omitempty"`
		StoreName  string          `json:"store_name,omitempty"`
		Templates  json.RawMessage `json:"templates,omitempty"`
	}

	ts, err := t.cache.MarshalJSON()
	if err != nil {
		return err
	}

	dump := templates{
		ExportedAt: time.Now(),
		StoreType:  t.Type(),
		StoreName:  t.Name(),
		Templates:  json.RawMessage(ts),
	}

	o, err := json.Marshal(dump)
	if err != nil {
		return err
	}

	_, err = t.file.Write(o)
	if err != nil {
		return err
	}

	return nil
}

// Start implements manager.Runnable, to handle the lifecycle of the persistent cache
func (t *PersistentCache) Start(ctx context.Context) error {
	// start the underlying cache asynchronously first, Start(...) will block the goroutine
	go t.cache.Start(ctx)

	// do initialization in a function closure such that we can easily unlock the mutex
	// from any of the branches, even on error. This is still synchronous!
	err := func() error {
		// t.mu.Lock()
		defer t.mu.Unlock()
		// defer t.wg.Done()

		err := t.Prepare()
		if err != nil {
			return err
		}

		err = t.Initialize(ctx)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return err
	}

	// block until the root context is cancelled, e.g., by signaling
	<-ctx.Done()

	// perform shutdown with a separate context that cancels automatically after 5 seconds
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// TODO(zoomoid): Close() does currently not adhere to the context's deadline, i.e., it runs
	// til completion anyway
	if err := t.Close(shutdownCtx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("cancelled context before completing dump to file, %w", err)
		}
		return err
	}
	return nil
}
