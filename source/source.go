package source

import "io"

// Source is an interface used by the parser to parse replays
type Source interface {
	Reader() io.Reader
	Error() error
}
