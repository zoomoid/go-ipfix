package ipfix

type SetHeader struct {
	// 0 for TemplateSet, 1 for OptionsTemplateSet, and
	// 256-65535 for DataSet as TemplateId (thus uint16)
	Id uint16 `json:"id,omitempty"`

	Length uint16 `json:"length,omitempty"`
}
