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

// listType is the interface implemented by BasicList to inject dependencies via builder pattern
// at the FieldBuilder level. Particularly, as *BasicList.Decode requires looking up information
// elements from a FieldCache, the ListTypeBuilder provides the singular injector for a FieldCache
type listType interface {
	DataType

	// NewBuilder instantiates a new ListTypeBuilder to provide the injection/builder context
	NewBuilder() listTypeBuilder
}

// listTypeBuilder is the interface to implement for list types that require a FieldCache during
// decoding, like BasicList. Note that the original type, i.e. BasicList, needs to implement
// ListType to create this builder.
//
//  1. Call ListType.NewBuilder() to create the builder type
//  2. provide a FieldCache using listTypeBuilder.WithFieldManager(...)
//  3. complete the builder by calling listTypeBuilder.Complete(), creating a new DataTypeConstructor
//     that is decorated with the FieldCache provided earlier.
//
// This pattern also applies to TemplateListTypes, see below
type listTypeBuilder interface {
	WithFieldCache(FieldCache) listTypeBuilder
	Complete() DataTypeConstructor
}

// templateListType is the interfaces implemented by SubTemplateList and SubTemplateMultiList to
// inject dependencies via builder pattern at the FieldBuilder level.
//
// A TemplateListTypeBuilder allows for injecting a FieldCache and a TemplateCache, both of which
// are required by SubTemplateList's and SubTemplateMultiList's Decode method.
type templateListType interface {
	DataType

	NewBuilder() templateListeTypeBuilder
}

// templateListeTypeBuilder is the interface to implement for template list types that require a
// FieldCache and/or a TemplateCache during decoding, i.e., SubTemplateList and SubTemplateMultiList.
//
//  1. Call TemplateListType.NewBuilder() to create the builder type
//  2. provide a FieldCache or TemplateCache using the respective method
//  3. complete the builder by calling templateListeTypeBuilder.Complete(), creating a new DataTypeConstructor
//     that is decorated with the caches provided earlier.
type templateListeTypeBuilder interface {
	WithTemplateCache(TemplateCache) templateListeTypeBuilder
	WithFieldCache(FieldCache) templateListeTypeBuilder

	// WithObservationDomain binds an observation domain id to the builder such that the underlying data type
	// may only retrieve templates from the designated "namespace".
	//
	// Templates are generally only available in those observation domains, and the relevant value is contained
	// in the IPFIX packet header, which is not available in the buffer given to the DataType to decode list
	// contents from. In order to be able to retrieve templates from the correct namespace, we need to inject
	// the observation domain id at the DataTypeConstructor level
	WithObservationDomain(id uint32) templateListeTypeBuilder
	Complete() DataTypeConstructor
}
