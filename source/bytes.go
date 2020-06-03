package source

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// ByteSource is used when you have a slice of bytes that need parsing.
type ByteSource struct {
	reader *bufio.Reader
	error  error
}

func (byteSource *ByteSource) Error() error {
	return byteSource.error
}

func (byteSource *ByteSource) Reader() io.Reader {
	return byteSource.reader
}

// FromBytes returns a source for a byte slice
func FromBytes(data []byte) Source {
	source := &ByteSource{}

	if data == nil {
		source.error = errors.New("nil bytes found")
	} else {
		source.reader = bufio.NewReader(bytes.NewReader(data))
	}

	return source
}
