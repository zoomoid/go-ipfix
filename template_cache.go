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
	"time"

	"errors"
	"fmt"
	"strconv"
	"strings"
)

// TemplateCache stores templates observed in an IPFIX/Netflow stream of flow packets
//
// Caches have to implement a function to
// - add a template defined by its version and observation domain ID,
// - retrieve a template by its version, its observation domain ID, and its ID, and
// - get all templates currently stored in the cache as a map
//
// Caches do not have to perform active expiry, for this, use TemplateCacheWithTimeout.
type TemplateCache interface {
	// GetAll returns the map of all templates currently stored in the cache
	GetAll(ctx context.Context) map[TemplateKey]*Template

	// Get returns the template stored at a given key, or an error if not found
	Get(ctx context.Context, key TemplateKey) (*Template, error)

	// Add adds a template at a given key into the cache. It may return an error if
	// anything bad happened during addition
	Add(ctx context.Context, key TemplateKey, template *Template) error

	Delete(ctx context.Context, key TemplateKey) error

	// Name returns the name of the cache set at construction
	Name() string

	// Type returns the constant type of the Cache as string
	Type() string

	// Caches implement json.Marshaler to be serializable
	json.Marshaler
}

type StatefulTemplateCache interface {
	TemplateCache

	// Start starts a stateful template cache. This is for example used in caches requiring a stateful connection
	// to a database/KV store like the etcd addon.
	//
	// These start/stop semantics are "leftovers" from the asynchronous architecture of FlowPlane from which this
	// library was factored out from. They might be removed in future (breaking) updates, as state management is
	// generally not the task of this library and thus usage and surface of these methods should be little.
	//
	// Start's behavior is to block indefinitely during the lifecycle of the template cache, which means that it is
	// best used *deferred*, either directly via a goroutine or via a lifecycle management component that starts
	// objects implementing such Start hooks. This is useful if you have many moving (read: concurrent) parts and are
	// using channels to pass data betweeen those components. Examples for this are asynchronous Apache Kafka producers,
	// which is the original setup how the decoder is used in FlowPlane.
	Start(context.Context) error

	// Close tears down any stateful component of a template store. E.g., this is used in the persistent template
	// cache to write the templates to disk before shutting down.
	Close(context.Context) error
}

// CachesWithTimeout is the interface to be implemented by caches that periodically expire templates
// which is according to the IPFIX spec (but seemingly never implemented in any of the FOSS collectors)
type TemplateCacheWithTimeout interface {
	TemplateCache

	// SetTimeout should update the internal timeout duration after which templates expire.
	// Implementing caches MAY update existing template deadlines, but MUST calculate new deadlines
	// using the latest duration
	SetTimeout(time.Duration)
}

// TemplateCacheDriver is the interface to be provided by TemplateCaches that have side effects, such as persistent
// caches that write to files. Here, the TemplateCacheDriver interface provides functionality to e.g. close file handles
// or read from files, effectively a hook system that can be used to e.g. restore and dump templates.
type TemplateCacheDriver interface {
	StatefulTemplateCache

	// Prepare is a validator for implementors of Driver that can return an error
	Prepare() error

	// Initialize is used for running context-dependent pre-checks such as connecting to KV databases, or opening file handles
	Initialize(context.Context) error

	// Close is used for destructing the cache's resources, e.g., closing file handles, disconnecting from databases etc.
	Close(context.Context) error
}

type TemplateKey struct {
	ObservationDomainId uint32
	TemplateId          uint16
}

func NewKey(observationDomainId uint32, templateId uint16) TemplateKey {
	return TemplateKey{
		ObservationDomainId: observationDomainId,
		TemplateId:          templateId,
	}
}

const (
	templateKeySeparator string = "-"
)

func (k *TemplateKey) String() string {
	return fmt.Sprintf("%d%s%d", k.ObservationDomainId, templateKeySeparator, k.TemplateId)
}

func (k *TemplateKey) MarshalText() (text []byte, err error) {
	text = []byte(k.String())
	return
}

func (k *TemplateKey) Unmarshal(text string) (err error) {
	var observationDomainId uint32
	var templateId uint16

	key := strings.Split(text, templateKeySeparator)
	if len(key) != 2 {
		return errors.New("template key format is invalid")
	}

	if v, err := strconv.ParseUint(key[0], 10, 64); err != nil {
		return fmt.Errorf("observation domain id is invalid, %w", err)
	} else {
		observationDomainId = uint32(v)
	}
	if v, err := strconv.ParseUint(key[1], 10, 64); err != nil {
		return fmt.Errorf("template id is invalid, %w", err)
	} else {
		templateId = uint16(v)
	}

	k.ObservationDomainId = observationDomainId
	k.TemplateId = templateId
	return
}

func (k *TemplateKey) UnmarshalText(text []byte) (err error) {
	return k.Unmarshal(string(text))
}
