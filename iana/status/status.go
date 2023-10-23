package status

type Status int

const (
	Current Status = iota

	Deprecated

	Obsolete

	Undefined
)

var supportedStatuses []Status = []Status{
	Current,
	Deprecated,
	Obsolete,
	Undefined,
}

func SupportedStatuses() []Status {
	return supportedStatuses
}

func (s Status) String() string {
	switch s {
	case Current:
		return "current"
	case Deprecated:
		return "deprecated"
	case Undefined:
		return "undefined"
	case Obsolete:
		return "obsolete"
	default:
		return ""
	}
}

func (s Status) MarshalText() ([]byte, error) {
	o := []byte(s.String())
	return o, nil
}

func (s *Status) UnmarshalText(in []byte) error {
	*s = Parse(string(in))
	return nil
}

func Parse(status string) Status {
	switch status {
	case "current":
		return Current
	case "deprecated":
		return Deprecated
	case "obsolete":
		return Obsolete
	default:
		return Undefined
	}
}
