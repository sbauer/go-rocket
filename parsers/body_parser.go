package parsers

import (
	"errors"
	"io"

	"github.com/sbauer/go-rocket/source"
)

type BodyParser struct {
	data     []byte
	reader   io.Reader
	position int32
	crc      uint32
	size     int32
}

type BodyParserResult struct {
}

func NewBodyParser(dataSource source.Source) *BodyParser {
	return &BodyParser{reader: dataSource.Reader()}
}

func (parser *BodyParser) Parse() (*BodyParserResult, error) {
	if parser.reader == nil {
		return nil, errors.New("Reader is nil")
	}

	parser.initializeBuffer()

	return nil, nil
}

func (parser *BodyParser) initializeBuffer() {

}

func (parser *BodyParser) take(numberOfBytes int) ([]byte, error) {

}
