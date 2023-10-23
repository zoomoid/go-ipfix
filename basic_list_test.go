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

import "testing"

func TestBasicList(t *testing.T) {

	t.Run("type guard for SetValue", func(t *testing.T) {

		sourceList := BasicList{
			semantic: SemanticOrdered,
			fieldId:  52,
			pen:      15151,
		}

		defer func() {
			recover()
		}()

		sourceList.SetValue([]DataType{
			&Boolean{
				value: true,
			},
			&Unsigned16{
				value: 4,
			},
			&Signed32{
				value: 12314,
			},
			&String{
				value: "abc",
			},
		})
	})
	t.Run("MarshalJSON", func(t *testing.T) {
		sourceList := BasicList{
			semantic: SemanticOrdered,
			fieldId:  52,
			pen:      15151,
		}

		sourceList.SetValue([]DataType{
			&Unsigned16{
				value: 1,
			},
			&Unsigned16{
				value: 2,
			},
			&Unsigned16{
				value: 4,
			},
			&Unsigned16{
				value: 8,
			},
			&Unsigned16{
				value: 16,
			},
		})
		o, err := sourceList.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(o))
	})
	t.Run("UnmarshalJSON", func(t *testing.T) {
		sourceList := BasicList{
			semantic: SemanticOrdered,
			fieldId:  52,
			pen:      15151,
		}

		sourceList.SetValue([]DataType{
			&Unsigned16{
				value: 1,
			},
			&Unsigned16{
				value: 2,
			},
			&Unsigned16{
				value: 4,
			},
			&Unsigned16{
				value: 8,
			},
			&Unsigned16{
				value: 16,
			},
		})
		o, err := sourceList.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		b := BasicList{}
		err = b.UnmarshalJSON(o)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(b.String())
	})
}
