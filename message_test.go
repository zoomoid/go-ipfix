package ipfix

import (
	"testing"
	"time"
)

func TestMessage_String(t *testing.T) {

	helloWorldField := &FixedLengthField{
		id:          5,
		pen:         12345,
		constructor: NewString,
		name:        "fixedString",
		value: &String{
			length: 11,
			value:  "hello world",
		},
	}

	msg := Message{
		Version:             10,
		Length:              16,
		ExportTime:          uint32(time.Now().Unix()),
		SequenceNumber:      1234,
		ObservationDomainId: 0,
		Sets: []Set{
			{
				SetHeader: SetHeader{
					Id:     3,
					Length: 8,
				},
				Kind: KindTemplateSet,
				Set: &TemplateSet{
					Records: []TemplateRecord{
						{
							FieldCount: 2,
							TemplateId: 1000,
							Fields: []Field{
								helloWorldField,
								&VariableLengthField{
									id:          6,
									name:        "stringBasicList",
									pen:         12345,
									constructor: NewBasicList,
									value: &BasicList{
										isVariableLength: true,
										semantic:         SemanticAllOf,
										pen:              12345,
										fieldId:          5,
										isEnterprise:     true,
										elementLength:    FieldVariableLength,
										length:           3,
										value: []Field{
											helloWorldField.Clone().SetValue("hello world 2"),
											helloWorldField.Clone().SetValue("hello world 3"),
											helloWorldField.Clone().SetValue("hello world 4"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	t.Log(msg.String())
	err := recover()
	if err != nil {
		t.Error(err)
	}
}
