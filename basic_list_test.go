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
