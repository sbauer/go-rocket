package source

import (
	"bufio"
	"bytes"
	"encoding/binary"
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

// ReadString parses the buffer for a string type
func (fileSource *FileSource) ReadString() (string, error) {
	var stringLength int32

	if stringLengthErr := binary.Read(fileSource.reader, binary.LittleEndian, &stringLength); stringLengthErr != nil {
		return "", stringLengthErr
	}

	stringBytes, err := fileSource.Read(int(stringLength))

	if err != nil {
		return "", err
	}

	stringBytes = bytes.Trim(stringBytes, "\x00")

	return string(stringBytes), nil
}

// Read reads a number of bytes from the buffer
func (fileSource *FileSource) Read(numberOfBytes int) ([]byte, error) {
	bytes := make([]byte, numberOfBytes)

	_, err := fileSource.reader.Read(bytes)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// ReadAsType reads data from the buffer and places it into a request type. This uses binary.Read internally
func (fileSource *FileSource) ReadAsType(interfaceType interface{}) error {
	return binary.Read(fileSource.reader, binary.LittleEndian, interfaceType)
}

// Error returns an error if present
func (fileSource *FileSource) Error() error {
	return fileSource.error
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
