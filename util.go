package ipfix

func IsEnterpriseField(fieldId uint16) bool {
	return fieldId>>15 == 1
}

func IsVariableLength(fieldLength uint16) bool {
	return fieldLength == 0xFFFF
}
