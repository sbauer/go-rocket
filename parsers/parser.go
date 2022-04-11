package parsers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

// Parser parses the replay file
type Parser struct {
	dataSource source.Source
}

// HeaderData is used to pull binary header data from the replay file
type HeaderData struct {
	HeaderSize      int32
	CRC             uint32
	EngineVersion   uint32
	LicenseeVersion uint32
}

// Parse processes the replay file and returns an instance of Replay
func Parse(dataSource source.Source) (*rocket.Replay, error) {
	parser := &Parser{dataSource: dataSource}

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

	parser.parseBody(reader, replay)
	replay.Properties = properties

	return replay, nil
}

func (parser *Parser) parseBody(reader io.Reader, replay *rocket.Replay) error {

	bodyParser := NewBodyParser(parser.dataSource)
	result, error := bodyParser.Parse()

	if error != nil {
		return error
	}

	replay.DebugData = result.debugData
	replay.Keyframes = result.keyframes
	replay.Levels = result.levels
	replay.Names = result.names
	replay.Objects = result.objects
	replay.Packages = result.packages
	replay.Tickmarks = result.tickmarks

	/*
		classIndexSize, _ := parser.ReadInt32(reader)
		netCacheSize, _ := parser.ReadInt32(reader)
	*/

	return nil
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
	name, initialError := readString(reader)

	if initialError != nil {
		return nil, initialError
	}

	if strings.Contains(name, "None") {
		return nil, nil
	}

	typeName, typeNameError := readString(reader)

	if typeNameError != nil {
		return nil, typeNameError
	}

	var (
		length      uint32
		unknownData uint32
		err         error
	)

	readAsType(reader, &length)
	readAsType(reader, &unknownData)

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

	readAsType(reader, &totalGroups)

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
		readAsType(reader, &intValue)
		return intValue, nil
	} else if strings.Contains(typeName, "Str") || strings.Contains(typeName, "Name") {
		value, err := readString(reader)

		if err != nil {
			return nil, err
		}
		return value, nil
	} else if strings.Contains(typeName, "Float") {
		var float float32
		readAsType(reader, &float)
		return float, nil
	} else if strings.Contains(typeName, "Byte") {
		var (
			first, second string
			err           error
		)

		first, err = readString(reader)

		if err != nil {
			return nil, err
		}

		second, err = readString(reader)

		if err != nil {
			return nil, err
		}

		return first + "|" + second, nil
	} else if strings.Contains(typeName, "Bool") {
		var byte byte
		readAsType(reader, &byte)
		convertedBool := byte != 0
		return convertedBool, nil
	} else if strings.Contains(typeName, "QWord") {
		var qword int64
		readAsType(reader, &qword)
		return qword, nil
	} else {
		return nil, errors.New("unknown property type: " + typeName)
	}
}

func (parser *Parser) parseHeader(reader io.Reader, replay *rocket.Replay) error {
	headerInfo := &HeaderData{}
	var netVersion uint32

	err := readAsType(reader, headerInfo)

	if err != nil {
		return err
	}

	if headerInfo.supportsNetVersion() {
		if netVersionErr := readAsType(reader, &netVersion); netVersionErr != nil {

		}
	}

	var className string

	if className, err = readString(reader); err != nil {
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

// ReadString parses the buffer for a string type. Currently does not understand encoding
func readString(reader io.Reader) (string, error) {
	stringLength, _ := readInt32(reader)

	stringBytes, err := read(reader, int(stringLength))

	if err != nil {
		fmt.Println(err)
		return "", err
	}

	output := string(stringBytes)

	return output, nil
}

// Read reads a number of bytes from the buffer
func read(reader io.Reader, numberOfBytes int) ([]byte, error) {
	bytes := make([]byte, numberOfBytes)

	_, err := io.ReadFull(reader, bytes)

	if err != nil {
		fmt.Println("read error", err)
		return nil, err
	}

	return bytes, nil
}

// ReadAsType reads data from the buffer and places it into a request type. This uses binary.Read internally
func readAsType(reader io.Reader, interfaceType interface{}) error {
	return binary.Read(reader, binary.LittleEndian, interfaceType)
}

func readInt32(reader io.Reader) (int32, error) {
	var (
		data int32
	)
	error := readAsType(reader, &data)

	return data, error
}

func readUInt32(reader io.Reader) (uint32, error) {
	var (
		data uint32
	)
	error := readAsType(reader, &data)

	return data, error
}

func readFloat32(reader io.Reader) (float32, error) {
	var (
		data float32
	)
	error := readAsType(reader, &data)

	return data, error
}

func parseStringList(reader io.Reader) ([]string, error) {
	listSize, error := readInt32(reader)

	if error != nil {
		return nil, error
	}

	list := make([]string, 0, listSize)

	for index := 0; index < int(listSize); index++ {
		objectName, error := readString(reader)

		if error != nil {
			return nil, error
		}

		list = append(list, objectName)
	}

	return list, nil
}
