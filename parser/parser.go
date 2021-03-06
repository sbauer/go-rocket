package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
	"io"
	"strings"
)

// Parser parses the replay file
type Parser struct {
}

// HeaderData is used to pull binary header data from the replay file
type HeaderData struct {
	HeaderSize      uint32
	CRC             uint32
	EngineVersion   uint32
	LicenseeVersion uint32
}

// Parse processes the replay file and returns an instance of Replay
func Parse(dataSource source.Source) (*rocket.Replay, error) {

	parser := &Parser{}

	if error := dataSource.Error(); error != nil {
		return nil, error
	}

	replay, err := parser.parseFile(dataSource.Reader())

	return replay, err
}

func (parser *Parser) parseFile(reader io.Reader) (*rocket.Replay, error) {
	replay := rocket.NewReplay()

	parser.parseHeader(reader, replay)

	properties, err := parser.parseProperties(reader, replay)

	if err != nil {
		return nil, err
	}

	replay.Properties = properties

	return replay, nil
}

func (parser *Parser) parseProperties(reader io.Reader, replay *rocket.Replay) (map[string]*rocket.Property, error) {
	properties := make(map[string]*rocket.Property)

	for {
		prop, err := parser.parseProperty(reader, replay)

		if err != nil {
			return nil, err
		}

		if prop == nil {
			break
		}

		properties[prop.Name] = prop
	}

	return properties, nil
}

func (parser *Parser) parseProperty(reader io.Reader, replay *rocket.Replay) (*rocket.Property, error) {
	name, initialError := parser.ReadString(reader)

	if initialError != nil {
		return nil, initialError
	}

	if strings.Contains(name, "None") {
		return nil, nil
	}

	typeName, typeNameError := parser.ReadString(reader)

	if typeNameError != nil {
		return nil, typeNameError
	}

	var (
		length      uint32
		unknownData uint32
		err         error
	)

	parser.ReadAsType(reader, &length)
	parser.ReadAsType(reader, &unknownData)

	prop := &rocket.Property{
		Name: name,
		Type: typeName,
	}

	if strings.Contains(typeName, "Array") {
		prop.Groups, err = parser.parsePropertyGroups(reader, replay)
	} else {
		prop.Value, err = parser.parsePropertyValue(reader, typeName)
	}

	return prop, err
}

func (parser *Parser) parsePropertyGroups(reader io.Reader, replay *rocket.Replay) ([]*rocket.PropertyGroup, error) {
	var totalGroups int32
	var err error

	parser.ReadAsType(reader, &totalGroups)

	groups := make([]*rocket.PropertyGroup, totalGroups)
	for index := int32(0); index < totalGroups; index++ {
		group := &rocket.PropertyGroup{}
		group.Properties, err = parser.parseProperties(reader, replay)
		groups[index] = group
	}

	return groups, err
}

// TODO: Refactor after figuring out format
func (parser *Parser) parsePropertyValue(reader io.Reader, typeName string) (interface{}, error) {
	if strings.Contains(typeName, "Int") {
		var intValue int32
		parser.ReadAsType(reader, &intValue)
		return intValue, nil
	} else if strings.Contains(typeName, "Str") || strings.Contains(typeName, "Name") {
		value, err := parser.ReadString(reader)

		if err != nil {
			return nil, err
		}
		return value, nil
	} else if strings.Contains(typeName, "Float") {
		var float float32
		parser.ReadAsType(reader, &float)
		return float, nil
	} else if strings.Contains(typeName, "Byte") {
		var (
			first, second string
			err           error
		)

		first, err = parser.ReadString(reader)

		if err != nil {
			return nil, err
		}

		second, err = parser.ReadString(reader)

		if err != nil {
			return nil, err
		}

		return first + "|" + second, nil
	} else if strings.Contains(typeName, "Bool") {
		var byte byte
		parser.ReadAsType(reader, &byte)
		convertedBool := byte != 0
		return convertedBool, nil
	} else if strings.Contains(typeName, "QWord") {
		var qword int64
		parser.ReadAsType(reader, &qword)
		return qword, nil
	} else {
		return nil, errors.New("unknown property type: " + typeName)
	}
}

func (parser *Parser) parseHeader(reader io.Reader, replay *rocket.Replay) error {
	headerInfo := &HeaderData{}
	var netVersion uint32

	err := parser.ReadAsType(reader, headerInfo)

	if err != nil {
		return err
	}

	if headerInfo.supportsNetVersion() {
		if netVersionErr := parser.ReadAsType(reader, &netVersion); netVersionErr != nil {

		}
	}

	var className string

	if className, err = parser.ReadString(reader); err != nil {
		return err
	}

	replay.HeaderInfo = &rocket.ReplayHeader{
		HeaderSize:      headerInfo.HeaderSize,
		CRC:             headerInfo.CRC,
		EngineVersion:   int(headerInfo.EngineVersion),
		LicenseeVersion: int(headerInfo.LicenseeVersion),
		NetVersion:      int(netVersion),
		ClassName:       className,
	}

	return nil
}

func (info *HeaderData) supportsNetVersion() bool {
	return info.EngineVersion >= 868 && info.LicenseeVersion >= 18
}

// ReadString parses the buffer for a string type
func (parser *Parser) ReadString(reader io.Reader) (string, error) {
	var stringLength int32

	if stringLengthErr := binary.Read(reader, binary.LittleEndian, &stringLength); stringLengthErr != nil {
		return "", stringLengthErr
	}

	stringBytes, err := parser.Read(reader, int(stringLength))

	if err != nil {
		return "", err
	}

	stringBytes = bytes.Trim(stringBytes, "\x00")

	return string(stringBytes), nil
}

// Read reads a number of bytes from the buffer
func (parser *Parser) Read(reader io.Reader, numberOfBytes int) ([]byte, error) {
	bytes := make([]byte, numberOfBytes)

	_, err := reader.Read(bytes)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// ReadAsType reads data from the buffer and places it into a request type. This uses binary.Read internally
func (parser *Parser) ReadAsType(reader io.Reader, interfaceType interface{}) error {
	return binary.Read(reader, binary.LittleEndian, interfaceType)
}
