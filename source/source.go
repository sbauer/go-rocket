package source

// Source is an interface used by the parser to parse replays
type Source interface {
	ReadString() (string, error)
	Read(numberOfBytes int) ([]byte, error)
	ReadAsType(interface{}) error
	Error() error
}
