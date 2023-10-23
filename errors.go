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

	"github.com/zoomoid/go-ipfix/iana/version"
)

var (
	ErrTemplateNotFound error = errors.New("template not found")
	ErrUnknownVersion   error = errors.New("unknown version")
	ErrUnknownFlowId    error = errors.New("unknown flow id")
)

func TemplateNotFound(observationDomainId uint32, templateId uint16) error {
	return fmt.Errorf("%w for %d in observation domain %d", ErrTemplateNotFound, templateId, observationDomainId)
}

func UnknownVersion(version version.ProtocolVersion) error {
	return fmt.Errorf("%w %d, only 9 and 10 are specified", ErrUnknownVersion, version)
}

func UnknownFlowId(id uint16) error {
	return fmt.Errorf("%w %d", ErrUnknownFlowId, id)
}
