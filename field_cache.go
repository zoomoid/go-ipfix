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
	"strconv"
	"strings"
	"sync"
)

var (
	ErrUnknownProtocolVersion  = errors.New("unknown protocol version in field manager")
	ErrUnknownEnterpriseNumber = errors.New("unknown enterprise number in field manager")
)

// FieldCache is the interface that all, both ephemeral and persistent field caches need to implement.
// By default, this does not include methods for handling stateful FieldCaches, those should be provided
// on the explicit types. See etcd.FieldCache for such an implementation.
type FieldCache interface {
	// GetBuilder retrieves a field builder instance from the cache for creating
	// fields during decoding.
	//
	// If the field is not found in the cache, a new UnassignedFieldBuilder is
	// returned with the information embedded in the FieldKey
	//
	// If an error occurs during retrieval of the field, an error is returned,
	// and the FieldBuilder pointer is nil
	GetBuilder(context.Context, FieldKey) (*FieldBuilder, error)

	// Add adds a new Information Element definition to the field cache.
	//
	// The canonic implementation of FieldCache immediately creates FieldBuilder
	// instances to return on Get(), however this is up to implementor.
	//
	// If adding the new IE fails, an error is returned.
	Add(context.Context, InformationElement) error

	// Delete removes a field identified by a FieldKey from the cache.
	//
	// The canonic implementation of FieldCache stores both information elements given during Add(),
	// and the instantiated FieldBuilder types, and cleans up both at once.
	Delete(context.Context, FieldKey) error

	// Get returns the information element that defines a field currently in the cache.
	//
	// Get returns an error if no element with the FieldKey is stored in the cache.
	//
	// Get returns errors that occur during retrieval of the information element.
	Get(context.Context, FieldKey) (*InformationElement, error)

	// GetAll returns a map of FieldBuilders for all fields currently stored in the cache.
	// If no fields are stored in the cache, the map is empty.
	GetAllBuilders(context.Context) map[FieldKey]*FieldBuilder

	// GetAll returns a map of InformationElements of all the fields stored in the cache.
	// If no information elements were added to the cache prior to the call, the map is empty.
	GetAll(context.Context) map[FieldKey]*InformationElement

	json.Marshaler
}

type FieldKey struct {
	EnterpriseId uint32
	Id           uint16
}

func NewFieldKey(enterpriseId uint32, fieldId uint16) FieldKey {
	return FieldKey{
		EnterpriseId: enterpriseId,
		Id:           fieldId,
	}
}

const (
	FieldKeySeparator string = ":"
)

func (k *FieldKey) String() string {
	return fmt.Sprintf("%d%s%d", k.EnterpriseId, FieldKeySeparator, k.Id)
}

func (k *FieldKey) MarshalText() (text []byte, err error) {
	text = []byte(k.String())
	return
}

func (k *FieldKey) Unmarshal(text string) (err error) {
	var enterpriseId uint32
	var fieldId uint16

	key := strings.Split(text, FieldKeySeparator)
	if len(key) != 2 {
		return errors.New("template key format is invalid")
	}

	if v, err := strconv.ParseUint(key[0], 10, 64); err != nil {
		return fmt.Errorf("observation domain id is invalid, %w", err)
	} else {
		enterpriseId = uint32(v)
	}
	if v, err := strconv.ParseUint(key[1], 10, 64); err != nil {
		return fmt.Errorf("template id is invalid, %w", err)
	} else {
		fieldId = uint16(v)
	}

	k.EnterpriseId = enterpriseId
	k.Id = fieldId
	return
}

func (k *FieldKey) UnmarshalText(text []byte) (err error) {
	return k.Unmarshal(string(text))
}

type EphemeralFieldCache struct {
	templateManager TemplateCache

	mu *sync.RWMutex

	fields map[FieldKey]*FieldBuilder

	prototypes map[FieldKey]*InformationElement
}

var _ json.Marshaler = &EphemeralFieldCache{}

func NewEphemeralFieldCache(templateManager TemplateCache) FieldCache {
	fm := &EphemeralFieldCache{
		mu: &sync.RWMutex{},
		// initialize an empty map of field builders
		fields:          map[FieldKey]*FieldBuilder{},
		prototypes:      map[FieldKey]*InformationElement{},
		templateManager: templateManager,
	}

	return fm
}

func (fm *EphemeralFieldCache) GetBuilder(ctx context.Context, key FieldKey) (*FieldBuilder, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	field, ok := fm.fields[key]
	if !ok {
		// logger.V(2).Info("fieldManager: unknown key", "enterpriseId", enterpriseId)
		return NewUnassignedFieldBuilder(key.Id).SetPEN(key.EnterpriseId), nil
	}
	return field, nil
}

func (fm *EphemeralFieldCache) Get(ctx context.Context, key FieldKey) (*InformationElement, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	ie, ok := fm.prototypes[key]
	if !ok {
		// logger.V(2).Info("fieldManager: unknown key", "enterpriseId", enterpriseId)
		return nil, fmt.Errorf("unknown information element for \"%s\"", key.String())
	}
	return ie, nil
}

func (fm *EphemeralFieldCache) Add(ctx context.Context, element InformationElement) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fk := NewFieldKey(element.EnterpriseId, element.Id)

	fm.prototypes[fk] = &element
	fm.fields[fk] = NewFieldBuilder(element).
		SetFieldManager(fm).
		SetTemplateManager(fm.templateManager).
		SetPEN(element.EnterpriseId)

	return nil
}

func (fm *EphemeralFieldCache) Delete(ctx context.Context, key FieldKey) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	delete(fm.fields, key)
	delete(fm.prototypes, key)
	return nil
}

func (fm *EphemeralFieldCache) GetAllBuilders(ctx context.Context) map[FieldKey]*FieldBuilder {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	return fm.fields
}

func (fm *EphemeralFieldCache) GetAll(ctx context.Context) map[FieldKey]*InformationElement {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	return fm.prototypes
}

func (fm *EphemeralFieldCache) MarshalJSON() ([]byte, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	s := make(map[string]interface{})
	for k, v := range fm.fields {
		s[k.String()] = v
	}
	return json.Marshal(s)
}

// newIPFIXFieldManager is a utility for creating field managers with initialized IANA fields quickly,
// e.g. for unit testing.
//
// newIPFIXFieldManager panics if failing to add an IE to the cache.
func newIPFIXFieldManager(templateManager TemplateCache) FieldCache {
	fm := NewEphemeralFieldCache(templateManager)
	for idx, ie := range IANA() {
		err := fm.Add(context.Background(), ie)
		if err != nil {
			panic(fmt.Errorf("failed to add IANA IE %d to ipfix field manager, %w", idx, err))
		}
	}
	return fm
}
