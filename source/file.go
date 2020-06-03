package source

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
)

// FileSource represents a replay from a file system
type FileSource struct {
	reader   *bufio.Reader
	fileName string
	valid    bool
	file     os.File
	error    error
}

// Error returns an error if present
func (fileSource *FileSource) Error() error {
	return fileSource.error
}

func (fileSource *FileSource) Reader() io.Reader {
	return fileSource.reader
}

// FromFile returns a source for a requested file
func FromFile(fileName string) Source {
	source := &FileSource{fileName: fileName}
	file, err := os.Open(fileName)

	if err != nil {
		source.error = err
	} else {
		data, readError := ioutil.ReadAll(file)

		if readError != nil {
			source.error = readError
		}

		source.reader = bufio.NewReader(bytes.NewReader(data))
	}

	return source
}
