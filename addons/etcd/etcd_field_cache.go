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

type FieldCache struct {
	client *clientv3.Client

	mu *sync.RWMutex

	templateCache ipfix.TemplateCache

	cache ipfix.FieldCache

	revisions map[ipfix.FieldKey]int64

	namespace string
	name      string
	prefix    string

	ready bool
}

var _ ipfix.FieldCache = &FieldCache{}

func NewDefaultFieldCache(client *clientv3.Client, fieldCache ipfix.FieldCache, templateCache ipfix.TemplateCache) *FieldCache {
	return NewNamedFieldCache("default", client, fieldCache, templateCache)
}

func NewNamedFieldCache(name string, client *clientv3.Client, fieldCache ipfix.FieldCache, templateCache ipfix.TemplateCache) *FieldCache {
	ns := "fields"
	prefix := ns + "/"

	client.KV = namespace.NewKV(client.KV, prefix)
	client.Watcher = namespace.NewWatcher(client.Watcher, prefix)
	client.Lease = namespace.NewLease(client.Lease, prefix)

	cache := &FieldCache{
		client:        client,
		templateCache: templateCache,
		mu:            &sync.RWMutex{},
		cache:         ipfix.NewEphemeralFieldCache(templateCache),
		revisions:     make(map[ipfix.FieldKey]int64),
		ready:         false,
		// TODO(zoomoid): logging currently isn't well-defined throughout this package
		// logger: nil,
		namespace: ns,
		name:      name,
		prefix:    name + "/",
	}

	cache.mu.Lock()
	return cache
}

func (f *FieldCache) GetBuilder(ctx context.Context, key ipfix.FieldKey) (*ipfix.FieldBuilder, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.cache.GetBuilder(ctx, key)
}

func (f *FieldCache) Get(ctx context.Context, key ipfix.FieldKey) (*ipfix.InformationElement, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.cache.Get(ctx, key)
}

func (f *FieldCache) Add(ctx context.Context, ie ipfix.InformationElement) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := ipfix.FieldKey{
		EnterpriseId: ie.EnterpriseId,
		Id:           ie.Id,
	}

	var txErr error
	defer func() {
		if txErr != nil {
			// rollback internal template addition
			f.cache.Delete(ctx, key)
		}
	}()

	err := f.cache.Add(ctx, ie)
	if err != nil {
		return err
	}

	_, txErr = f.put(ctx, key, &ie)
	if txErr != nil {
		return txErr
	}

	f.revisions[key] += 1

	return nil
}

func (f *FieldCache) GetAllBuilders(ctx context.Context) map[ipfix.FieldKey]*ipfix.FieldBuilder {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.cache.GetAllBuilders(ctx)
}

func (f *FieldCache) GetAll(ctx context.Context) map[ipfix.FieldKey]*ipfix.InformationElement {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.cache.GetAll(ctx)
}

func (f *FieldCache) Delete(ctx context.Context, key ipfix.FieldKey) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	defer delete(f.revisions, key)
	return f.cache.Delete(ctx, key)
}

func (f *FieldCache) Name() string {
	return fmt.Sprintf("%s/%s", f.namespace, f.name)
}

func (f *FieldCache) Type() string {
	return fmt.Sprintf("%s/%s", "etcd", "field")
}

func (f *FieldCache) MarshalJSON() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	type ifs struct {
		Type  string          `json:"type,omitempty"`
		Name  string          `json:"name,omitempty"`
		Cache json.RawMessage `json:"cache,omitempty"`
	}

	cc, err := f.cache.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return json.Marshal(ifs{
		Type:  f.Type(),
		Name:  f.Name(),
		Cache: cc,
	})
}

func (f *FieldCache) Start(ctx context.Context) error {
	logger := ipfix.FromContext(ctx)

	err := func() error {
		// restore from etcd shard
		defer f.mu.Unlock()

		logger.V(2).Info("initializing template cache from etcd")
		err := f.initialize(ctx)
		if err != nil {
			return err
		}

		return nil
	}()
	if err != nil {
		return err
	}

	go f.sync(ctx)

	<-ctx.Done()

	if err := f.client.Close(); err != nil {
		return err
	}
	return nil
}

func (f *FieldCache) initialize(ctx context.Context) error {
	// read any pre-existing fields from etcd
	res, err := f.client.Get(ctx, f.prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return err
	}

	fieldMap := make(map[ipfix.FieldKey]ipfix.InformationElement)
	for _, e := range res.Kvs {
		ie := ipfix.InformationElement{}
		err = json.Unmarshal(e.Value, &ie)
		if err != nil {
			return err
		}
		kkey := ipfix.FieldKey{}
		err = kkey.UnmarshalText(e.Key)
		if err != nil {
			return err
		}
		fieldMap[kkey] = ie
		f.revisions[kkey] = e.Version
	}
	for _, v := range fieldMap {
		// directly add the template to the underlying in-memory cache
		err := f.cache.Add(ctx, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FieldCache) sync(ctx context.Context) {
	logger := ipfix.FromContext(ctx)

	rch := f.client.Watch(ctx, f.prefix, clientv3.WithPrefix())
	for {
		select {
		case ev := <-rch:
			err := f.updateLocalFields(ctx, ev.Events)
			if err != nil {
				logger.Error(err, "failed to update internal field cache from watch event")
			}
			logger.V(2).Info("completed sync cycle for etcd fields")
		case <-ctx.Done():
			return
		}
	}
}

func (f *FieldCache) updateLocalFields(ctx context.Context, events []*clientv3.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, e := range events {
		element := e.Kv

		kkey := strings.TrimPrefix(string(element.Key), f.prefix)
		key := ipfix.FieldKey{}
		err := key.Unmarshal(kkey)
		if err != nil {
			return err
		}

		if prevRev, ok := f.revisions[key]; ok && prevRev < element.Version {
			ie := ipfix.InformationElement{}
			err := json.Unmarshal(element.Value, &ie)
			if err != nil {
				return err
			}
			err = f.cache.Add(ctx, ie)
			if err != nil {
				return err
			}
			f.revisions[key] = element.Version
		}
	}
	return nil
}

func (f *FieldCache) put(ctx context.Context, key ipfix.FieldKey, ie *ipfix.InformationElement) (*clientv3.PutResponse, error) {
	etcdKey := f.prefix + key.String()
	eei, err := json.Marshal(ie)
	if err != nil {
		return nil, err
	}

	return f.client.Put(ctx, etcdKey, string(eei))
}
