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
	"errors"
	"fmt"
)

var (
	// ErrTemplateNotFound is the base error used for indicating missing templates in caches.
	// It may be used in errors.Is() checks for error type, whereas compound errors constructed
	// with TemplateNotFound(...) cannot be compared with == due to including more information
	ErrTemplateNotFound error = errors.New("template not found")
	// ErrUnknownVersion indicates an illegal version number for IPFIX in the header of the message.
	ErrUnknownVersion error = errors.New("unknown version")
	// ErrUnknownFlowId is used for indicating usage of a set ID unassigned in IPFIX, which is specifically
	// the interval [5, 255], which is reserved.
	ErrUnknownFlowId error = errors.New("unknown flow id")

	// ErrIllegalDataTypeEncoding is used in Decode of certain data types that explicitly define illegal formats
	// such as boolean (1 and 2 encoding true and false and all other values being illegal) or strings
	// only allowing utf8 sequences.
	ErrIllegalDataTypeEncoding = errors.New("illegal data type encoding")
)

// templateNotFound wraps ErrTemplateNotFound to provide more information about _where_ the template
// was expected to be
func templateNotFound(observationDomainId uint32, templateId uint16) error {
	return fmt.Errorf("%w for %d in observation domain %d", ErrTemplateNotFound, templateId, observationDomainId)
}
