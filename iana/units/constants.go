package units

const (
	None           string = "none"
	Bits           string = "bits"
	Octets         string = "octets"
	Packets        string = "packets"
	Flows          string = "flows"
	Seconds        string = "seconds"
	Milliseconds   string = "milliseconds"
	Microseconds   string = "microseconds"
	Nanoseconds    string = "nanoseconds"
	FourOctetWords string = "4-octet-words"
	Messages       string = "messages"
	Hops           string = "hops"
	Entries        string = "entries"
	Frames         string = "frames"
	Ports          string = "ports"
	Inferred       string = "inferred"
	Unassigned     string = "unassigned"
)

func FromNumber(i uint16) string {
	switch i {
	case 0:
		return None
	case 1:
		return Bits
	case 2:
		return Octets
	case 3:
		return Packets
	case 4:
		return Flows
	case 5:
		return Seconds
	case 6:
		return Milliseconds
	case 7:
		return Microseconds
	case 8:
		return Nanoseconds
	case 9:
		return FourOctetWords
	case 10:
		return Messages
	case 11:
		return Hops
	case 12:
		return Entries
	case 13:
		return Frames
	case 14:
		return Ports
	case 16:
		return Inferred
	default:
		return Unassigned
	}
}
