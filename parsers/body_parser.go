package parsers

import (
	"bytes"
	"errors"
	"io"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

type BodyParser struct {
	data       []byte
	reader     io.Reader
	position   int32
	crc        uint32
	size       int32
	dataSource source.Source
	verifyCrc  bool
	actualCrc  uint32
}

type BodyParserResult struct {
	levels      []string
	keyframes   []*rocket.Keyframe
	networkData []byte
	debugData   []*rocket.DebugData
	tickmarks   []*rocket.Tickmark
	packages    []string
	objects     []string
	names       []string
}

func NewBodyParser(dataSource source.Source) *BodyParser {

	if dataSource == nil {
		return nil
	}

	return &BodyParser{dataSource: dataSource, verifyCrc: true}
}

func (parser *BodyParser) Parse() (*BodyParserResult, error) {
	if parser.dataSource == nil {
		return nil, errors.New("Source is nil")
	}

	parser.initializeBufferAndVerifyIfNeeded()

	levels, error := parseStringList(parser.reader)

	if error != nil {
		return nil, error
	}

	keyframes, error := parser.parseKeyFrames()

	if error != nil {
		return nil, error
	}

	networkBuffer, error := parser.parseNetworkData()

	if error != nil {
		return nil, error
	}

	debugInfo, error := parser.parseDebugInfo()

	tickmarks, error := parser.parseTickmarks()

	packages, error := parseStringList(parser.reader)

	if error != nil {
		return nil, error
	}

	objects, error := parseStringList(parser.reader)

	if error != nil {
		return nil, error
	}

	names, error := parseStringList(parser.reader)

	if error != nil {
		return nil, error
	}

	result := &BodyParserResult{
		levels:      levels,
		keyframes:   keyframes,
		networkData: networkBuffer,
		debugData:   debugInfo,
		tickmarks:   tickmarks,
		packages:    packages,
		objects:     objects,
		names:       names,
	}

	return result, nil
}

func (bp *BodyParser) parseTickmarks() ([]*rocket.Tickmark, error) {
	tickmarkSize, error := readInt32(bp.reader)

	if error != nil {
		return nil, error
	}

	tickmarks := make([]*rocket.Tickmark, 0, tickmarkSize)

	for index := 0; index < int(tickmarkSize); index++ {
		description, error := readString(bp.reader)
		if error != nil {
			return nil, error
		}

		frame, error := readInt32(bp.reader)
		if error != nil {
			return nil, error
		}

		tickmarks = append(tickmarks, &rocket.Tickmark{Frame: frame, Description: description})
	}

	return tickmarks, nil
}

func (bp *BodyParser) parseDebugInfo() ([]*rocket.DebugData, error) {
	debugInfoSize, error := readInt32(bp.reader)

	if error != nil {
		return nil, error
	}

	debugData := make([]*rocket.DebugData, 0, debugInfoSize)

	for index := 0; index < int(debugInfoSize); index++ {
		frame, error := readInt32(bp.reader)
		if error != nil {
			return nil, error
		}

		user, error := readString(bp.reader)
		if error != nil {
			return nil, error
		}

		text, error := readString(bp.reader)
		if error != nil {
			return nil, error
		}

		debugData = append(debugData, &rocket.DebugData{User: user, Frame: frame, Text: text})
	}

	return debugData, nil
}

func (bodyParser *BodyParser) parseNetworkData() ([]byte, error) {
	networkSize, _ := readInt32(bodyParser.reader)

	networkBuffer := make([]byte, networkSize)

	error := readAsType(bodyParser.reader, &networkBuffer)

	return networkBuffer, error
}

func (bodyParser *BodyParser) parseKeyFrames() ([]*rocket.Keyframe, error) {
	keyframeSize, error := readInt32(bodyParser.reader)

	keyframes := make([]*rocket.Keyframe, 0, keyframeSize)

	if error != nil {
		return nil, error
	}

	for index := 0; index < int(keyframeSize); index++ {
		time, error := readFloat32(bodyParser.reader)

		if error != nil {
			return nil, error
		}

		frame, error := readInt32(bodyParser.reader)

		if error != nil {
			return nil, error
		}
		position, error := readInt32(bodyParser.reader)

		if error != nil {
			return nil, error
		}

		keyframes = append(keyframes, &rocket.Keyframe{Time: time, Frame: frame, Position: position})
	}

	return keyframes, nil
}

func (bodyParser *BodyParser) initializeBufferAndVerifyIfNeeded() {
	bodyParser.size, _ = readInt32(bodyParser.dataSource.Reader())
	bodyParser.crc, _ = readUInt32(bodyParser.dataSource.Reader())

	data := make([]byte, bodyParser.size)

	readAsType(bodyParser.dataSource.Reader(), &data)

	if bodyParser.verifyCrc {

	}

	bodyParser.reader = bytes.NewBuffer(data)
}
