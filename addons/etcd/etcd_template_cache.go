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

package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/zoomoid/go-ipfix"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/namespace"
)

type TemplateCache struct {
	client *clientv3.Client

	mu *sync.RWMutex

	// fieldCache is required for injecting into TemplateRecords and
	// subsequently Fields during reconstruction from JSON
	fieldCache ipfix.FieldCache

	// cache is the in-memory cache used to cache idempotent operations
	// between etcd and the collector
	cache ipfix.StatefulTemplateCache

	// revision map is used to maintain template version in a distributed scenario:
	// When receiving either a new template record or redefining an existing template,
	// it will increment the revision count locally and PUT the update/create request to
	// etcd. All watchers will receive the revision through event channels and a mismatch
	// in revision will indicate the template cache to replace its local version of the
	// template with the new one.
	//
	// This is primarily done to prevent recursiveness in updating the templates. Note that
	// due to monotonicity of revisions, we can use ordered comparison operators rather than
	// just simple inequality.
	revisions map[ipfix.TemplateKey]int64

	namespace string
	name      string
	prefix    string

	ready bool
}

var _ ipfix.TemplateCache = &TemplateCache{}
var _ ipfix.TemplateCacheDriver = &TemplateCache{}

func NewDefaultTemplateCache(client *clientv3.Client, templateCache ipfix.StatefulTemplateCache, fieldCache ipfix.FieldCache) *TemplateCache {
	return NewNamedTemplateCache("default", client, templateCache, fieldCache)
}

func NewNamedTemplateCache(name string, client *clientv3.Client, templateCache ipfix.StatefulTemplateCache, fieldCache ipfix.FieldCache) *TemplateCache {
	ns := "templates"
	prefix := ns + "/"

	client.KV = namespace.NewKV(client.KV, prefix)
	client.Watcher = namespace.NewWatcher(client.Watcher, prefix)
	client.Lease = namespace.NewLease(client.Lease, prefix)

	cache := &TemplateCache{
		client:     client,
		cache:      templateCache,
		fieldCache: fieldCache,
		mu:         &sync.RWMutex{},
		revisions:  make(map[ipfix.TemplateKey]int64),
		ready:      false,

		// TODO(zoomoid): logging currently isn't well-defined throughout this package
		// logger: ,

		namespace: ns,
		name:      name,
		prefix:    name + "/",
	}
	cache.mu.Lock()
	return cache
}

func (t *TemplateCache) Add(ctx context.Context, key ipfix.TemplateKey, template *ipfix.Template) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var txErr error
	defer func() {
		if txErr != nil {
			// rollback internal template addition
			t.cache.Delete(ctx, key)
		}
	}()

	err := t.cache.Add(ctx, key, template)
	if err != nil {
		return err
	}

	_, txErr = t.put(ctx, key, template)
	if txErr != nil {
		return txErr
	}

	t.revisions[key] += 1

	return nil
}

func (t *TemplateCache) GetAll(ctx context.Context) map[ipfix.TemplateKey]*ipfix.Template {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.cache.GetAll(ctx)
}

func (t *TemplateCache) Get(ctx context.Context, key ipfix.TemplateKey) (*ipfix.Template, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.cache.Get(ctx, key)
}

func (t *TemplateCache) Delete(ctx context.Context, key ipfix.TemplateKey) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	defer delete(t.revisions, key)
	return t.cache.Delete(ctx, key)
}

func (t *TemplateCache) MarshalJSON() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// t.wg.Wait()

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

func (t *TemplateCache) Name() string {
	return fmt.Sprintf("%s/%s", t.namespace, t.name)
}

func (t *TemplateCache) Type() string {
	return fmt.Sprintf("%s/%s", "etcd", t.cache.Type())
}

func (t *TemplateCache) Prepare() error {
	return nil
}

// Initialize fetches all templates stored in etcd for a particular template space and
// reconstructs the internal map of templates
func (t *TemplateCache) Initialize(ctx context.Context) error {
	// read templates from etcd
	res, err := t.client.Get(ctx, t.prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return err
	}

	templateMap := make(map[ipfix.TemplateKey]*ipfix.Template)
	for _, e := range res.Kvs {
		tmpl := (&ipfix.Template{}).WithFieldCache(t.fieldCache).WithTemplateCache(t.cache)
		err = json.Unmarshal(e.Value, tmpl)
		if err != nil {
			return err
		}
		kkey := ipfix.TemplateKey{}
		err = kkey.UnmarshalText(e.Key)
		if err != nil {
			return err
		}
		templateMap[kkey] = tmpl
		t.revisions[kkey] = e.Version
	}
	for k, v := range templateMap {
		// directly add the template to the underlying in-memory cache
		err := t.cache.Add(ctx, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TemplateCache) Close(ctx context.Context) error {
	defer t.client.Close()
	defer t.cache.Close(ctx)

	return nil
}

func (t *TemplateCache) Start(ctx context.Context) error {
	logger := ipfix.FromContext(ctx)

	go t.cache.Start(ctx)
	err := func() error {
		defer t.mu.Unlock()

		err := t.Prepare()
		if err != nil {
			return err
		}
		logger.V(2).Info("initializing template cache from etcd")
		err = t.Initialize(ctx)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return err
	}

	go t.sync(ctx)

	<-ctx.Done()

	if err := t.client.Close(); err != nil {
		return err
	}
	return nil
}

// sync runs to receive updates from etcd about template creation and updates
func (t *TemplateCache) sync(ctx context.Context) {
	logger := ipfix.FromContext(ctx)
	rch := t.client.Watch(ctx, t.prefix, clientv3.WithPrefix())
	for {
		select {
		case ev := <-rch:
			err := t.updateLocalTemplates(ctx, ev.Events)
			if err != nil {
				logger.Error(err, "failed to update internal template cache from watch event")
			}
			logger.V(2).Info("completed sync cycle for etcd templates")
		case <-ctx.Done():
			return
		}
	}
}

func (t *TemplateCache) updateLocalTemplates(ctx context.Context, events []*clientv3.Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, e := range events {
		element := e.Kv

		// need to remove the key prefix here first...
		kkey := strings.TrimPrefix(string(element.Key), t.prefix)
		key := ipfix.TemplateKey{}
		err := key.Unmarshal(kkey)
		if err != nil {
			return err
		}

		if prevRev, ok := t.revisions[key]; ok && prevRev < element.Version {
			tmpl := (&ipfix.Template{}).WithFieldCache(t.fieldCache).WithTemplateCache(t.cache)
			err := json.Unmarshal(element.Value, tmpl)
			if err != nil {
				return err
			}
			err = t.cache.Add(ctx, key, tmpl)
			if err != nil {
				return err
			}
			t.revisions[key] = element.Version
		}
	}
	return nil
}

func (t *TemplateCache) put(ctx context.Context, key ipfix.TemplateKey, template *ipfix.Template) (*clientv3.PutResponse, error) {
	etcdKey := t.prefix + key.String()
	tmpl, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}

	return t.client.Put(ctx, etcdKey, string(tmpl))
}
