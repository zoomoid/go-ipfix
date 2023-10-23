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

type SetHeader struct {
	// 0 for TemplateSet, 1 for OptionsTemplateSet, and
	// 256-65535 for DataSet as TemplateId (thus uint16)
	Id uint16 `json:"id,omitempty"`

	Length uint16 `json:"length,omitempty"`
}
