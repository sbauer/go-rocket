package parser

import (
	"errors"
	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
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

	replay, err := parser.parseFile(dataSource)

	return replay, err
}

func (parser *Parser) parseFile(dataSource source.Source) (*rocket.Replay, error) {
	replay := rocket.NewReplay()

	parser.parseHeader(dataSource, replay)

	properties, err := parser.parseProperties(dataSource, replay)

	if err != nil {
		return nil, err
	}

	replay.Properties = properties

	return replay, nil
}

func (parser *Parser) parseProperties(dataSource source.Source, replay *rocket.Replay) (map[string]*rocket.Property, error) {
	properties := make(map[string]*rocket.Property)

	for {
		prop, err := parser.parseProperty(dataSource, replay)

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

func (parser *Parser) parseProperty(dataSource source.Source, replay *rocket.Replay) (*rocket.Property, error) {
	name, initialError := dataSource.ReadString()

	if initialError != nil {
		return nil, initialError
	}

	if strings.Contains(name, "None") {
		return nil, nil
	}

	typeName, typeNameError := dataSource.ReadString()

	if typeNameError != nil {
		return nil, typeNameError
	}

	var (
		length      uint32
		unknownData uint32
		err         error
	)

	dataSource.ReadAsType(&length)
	dataSource.ReadAsType(&unknownData)

	prop := &rocket.Property{
		Name: name,
		Type: typeName,
	}

	if strings.Contains(typeName, "Array") {
		prop.Groups, err = parser.parsePropertyGroups(dataSource, replay)
	} else {
		prop.Value, err = parser.parsePropertyValue(dataSource, typeName)
	}

	return prop, err
}

func (parser *Parser) parsePropertyGroups(dataSource source.Source, replay *rocket.Replay) ([]*rocket.PropertyGroup, error) {
	var totalGroups int32
	var err error

	dataSource.ReadAsType(&totalGroups)

	groups := make([]*rocket.PropertyGroup, totalGroups)
	for index := int32(0); index < totalGroups; index++ {
		group := &rocket.PropertyGroup{}
		group.Properties, err = parser.parseProperties(dataSource, replay)
		groups[index] = group
	}

	return groups, err
}

// TODO: Refactor after figuring out format
func (parser *Parser) parsePropertyValue(dataSource source.Source, typeName string) (interface{}, error) {
	if strings.Contains(typeName, "Int") {
		var intValue int32
		dataSource.ReadAsType(&intValue)
		return intValue, nil
	} else if strings.Contains(typeName, "Str") || strings.Contains(typeName, "Name") {
		value, err := dataSource.ReadString()

		if err != nil {
			return nil, err
		}
		return value, nil
	} else if strings.Contains(typeName, "Float") {
		var float float32
		dataSource.ReadAsType(&float)
		return float, nil
	} else if strings.Contains(typeName, "Byte") {
		var (
			first, second string
			err           error
		)

		first, err = dataSource.ReadString()

		if err != nil {
			return nil, err
		}

		second, err = dataSource.ReadString()

		if err != nil {
			return nil, err
		}

		return first + "|" + second, nil
	} else if strings.Contains(typeName, "Bool") {
		var byte byte
		dataSource.ReadAsType(&byte)
		convertedBool := byte != 0
		return convertedBool, nil
	} else if strings.Contains(typeName, "QWord") {
		var qword int64
		dataSource.ReadAsType(&qword)
		return qword, nil
	} else {
		return nil, errors.New("unknown property type: " + typeName)
	}
}

func (*Parser) parseHeader(dataSource source.Source, replay *rocket.Replay) error {
	headerInfo := &HeaderData{}
	var netVersion uint32

	err := dataSource.ReadAsType(headerInfo)

	if err != nil {
		return err
	}

	if headerInfo.supportsNetVersion() {
		if netVersionErr := dataSource.ReadAsType(&netVersion); netVersionErr != nil {

		}
	}

	var className string

	if className, err = dataSource.ReadString(); err != nil {
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
