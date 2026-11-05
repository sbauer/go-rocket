package parsers

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

// Parser parses the replay file
type Parser struct {
	dataSource    source.Source
	crcValidation CRCValidationMode
	decodeNetwork bool
	propertyDepth int
}

const maxPropertyDepth = 128

// CRCValidationMode controls when replay section CRCs are checked.
type CRCValidationMode uint8

const (
	// CRCValidationAlways checks CRCs before parsing each section.
	CRCValidationAlways CRCValidationMode = iota
	// CRCValidationNever skips CRC checks.
	CRCValidationNever
	// CRCValidationOnParseError checks a section CRC only if parsing that section fails.
	CRCValidationOnParseError
)

// ParseOptions configures replay parsing.
type ParseOptions struct {
	CRCValidation CRCValidationMode
	// DecodeNetwork decodes frames and actor replications. It is disabled by default.
	DecodeNetwork bool
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
	return ParseWithOptions(dataSource, ParseOptions{})
}

// ParseFile parses a replay from a file using the supplied options.
func ParseFile(fileName string, options ParseOptions) (*rocket.Replay, error) {
	return ParseWithOptions(source.FromFile(fileName), options)
}

// ParseWithOptions processes a replay file using the supplied options.
func ParseWithOptions(dataSource source.Source, options ParseOptions) (*rocket.Replay, error) {
	if options.CRCValidation > CRCValidationOnParseError {
		return nil, fmt.Errorf("invalid CRC validation mode: %d", options.CRCValidation)
	}
	if dataSource == nil {
		return nil, errors.New("source is nil")
	}

	if err := dataSource.Error(); err != nil {
		return nil, err
	}

	reader := dataSource.Reader()
	if reader == nil {
		return nil, errors.New("source reader is nil")
	}

	parser := &Parser{
		dataSource:    dataSource,
		crcValidation: options.CRCValidation,
		decodeNetwork: options.DecodeNetwork,
	}
	return parser.parseFile(reader)
}

func (parser *Parser) parseFile(reader io.Reader) (*rocket.Replay, error) {
	replay := rocket.NewReplay()

	headerSize, err := readCount(reader)
	if err != nil {
		return nil, fmt.Errorf("header size: %w", err)
	}

	headerCRC, err := readUInt32(reader)
	if err != nil {
		return nil, fmt.Errorf("header CRC: %w", err)
	}

	headerData, err := read(reader, int64(headerSize))
	if err != nil {
		return nil, fmt.Errorf("header data: %w", err)
	}
	if parser.crcValidation == CRCValidationAlways {
		if err := validateCRC("header", headerCRC, headerData); err != nil {
			return nil, err
		}
	}

	headerReader := bytes.NewReader(headerData)
	headerInfo := &HeaderData{HeaderSize: headerSize, CRC: headerCRC}
	if err := parser.parseHeader(headerReader, replay, headerInfo); err != nil {
		if crcErr := parser.headerCRCErrorOnParseFailure(headerCRC, headerData, err); crcErr != nil {
			return nil, crcErr
		}
		return nil, fmt.Errorf("header: %w", err)
	}

	properties, orderedProperties, err := parser.parseProperties(headerReader, replay)
	if err != nil {
		if crcErr := parser.headerCRCErrorOnParseFailure(headerCRC, headerData, err); crcErr != nil {
			return nil, crcErr
		}
		return nil, fmt.Errorf("header properties: %w", err)
	}

	replay.Properties = properties
	replay.OrderedProperties = orderedProperties
	if err := parser.parseBody(reader, replay); err != nil {
		return nil, fmt.Errorf("body: %w", err)
	}

	if parser.decodeNetwork {
		replay.Frames, err = decodeNetwork(replay)
		if err != nil {
			return nil, fmt.Errorf("network: %w", err)
		}
	}

	return replay, nil
}

func (parser *Parser) headerCRCErrorOnParseFailure(expected uint32, data []byte, parseErr error) error {
	if parser.crcValidation != CRCValidationOnParseError {
		return nil
	}
	if err := validateCRC("header", expected, data); err != nil {
		return fmt.Errorf("%w (header parsing failed: %v)", err, parseErr)
	}
	return nil
}

func (parser *Parser) parseBody(reader io.Reader, replay *rocket.Replay) error {
	bodyParser := NewBodyParser(parser.dataSource)
	bodyParser.crcValidation = parser.crcValidation
	result, err := bodyParser.parse(reader)

	if err != nil {
		return err
	}

	replay.DebugData = result.debugData
	replay.Keyframes = result.keyframes
	replay.Levels = result.levels
	replay.NetworkData = result.networkData
	replay.Names = result.names
	replay.Objects = result.objects
	replay.Packages = result.packages
	replay.Tickmarks = result.tickmarks
	replay.ClassIndices = result.classIndices
	replay.NetCache = result.netCache

	return nil
}

func (parser *Parser) parseProperties(reader io.Reader, replay *rocket.Replay) (map[string]*rocket.Property, []*rocket.Property, error) {
	if parser.propertyDepth >= maxPropertyDepth {
		return nil, nil, fmt.Errorf("property nesting exceeds %d levels", maxPropertyDepth)
	}
	parser.propertyDepth++
	defer func() { parser.propertyDepth-- }()

	properties := make(map[string]*rocket.Property)
	orderedProperties := make([]*rocket.Property, 0)

	for {
		prop, err := parser.parseProperty(reader, replay)

		if err != nil {
			return nil, nil, err
		}

		if prop == nil {
			break
		}

		properties[prop.Name] = prop
		orderedProperties = append(orderedProperties, prop)
	}

	return properties, orderedProperties, nil
}

func (parser *Parser) parseProperty(reader io.Reader, replay *rocket.Replay) (*rocket.Property, error) {
	name, initialError := readIdentifier(reader)

	if initialError != nil {
		return nil, initialError
	}

	if name == "None" {
		return nil, nil
	}

	typeName, typeNameError := readIdentifier(reader)

	if typeNameError != nil {
		return nil, fmt.Errorf("property %q type: %w", name, typeNameError)
	}

	var (
		length      uint32
		unknownData uint32
		err         error
	)

	if err := readAsType(reader, &length); err != nil {
		return nil, fmt.Errorf("property %q size: %w", name, err)
	}
	if err := readAsType(reader, &unknownData); err != nil {
		return nil, fmt.Errorf("property %q index: %w", name, err)
	}

	prop := &rocket.Property{
		Name: name,
		Type: typeName,
	}

	switch typeName {
	case "ArrayProperty":
		prop.Groups, err = parser.parsePropertyGroups(reader, replay)
	case "StructProperty":
		var structName string
		structName, err = readIdentifier(reader)
		if err == nil {
			var fields map[string]*rocket.Property
			var orderedFields []*rocket.Property
			fields, orderedFields, err = parser.parseProperties(reader, replay)
			if err == nil {
				prop.Value = rocket.StructProperty{Name: structName, Fields: fields, OrderedFields: orderedFields}
			}
		}
	default:
		legacy := replay != nil && replay.HeaderInfo != nil &&
			replay.HeaderInfo.EngineVersion == 0 && replay.HeaderInfo.LicenseeVersion == 0
		prop.Value, err = parser.parsePropertyValue(reader, typeName, legacy)
	}

	if err != nil {
		return nil, fmt.Errorf("property %q: %w", name, err)
	}

	return prop, nil
}

func (parser *Parser) parsePropertyGroups(reader io.Reader, replay *rocket.Replay) ([]*rocket.PropertyGroup, error) {
	totalGroups, err := readCount(reader)
	if err != nil {
		return nil, err
	}

	groups := make([]*rocket.PropertyGroup, 0)
	for index := int32(0); index < totalGroups; index++ {
		group := &rocket.PropertyGroup{}
		group.Properties, group.OrderedProperties, err = parser.parseProperties(reader, replay)
		if err != nil {
			return nil, fmt.Errorf("group %d: %w", index, err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (parser *Parser) parsePropertyValue(reader io.Reader, typeName string, legacy bool) (interface{}, error) {
	switch typeName {
	case "IntProperty":
		var intValue int32
		err := readAsType(reader, &intValue)
		return intValue, err
	case "StrProperty", "NameProperty":
		return readString(reader)
	case "FloatProperty":
		var float float32
		err := readAsType(reader, &float)
		return float, err
	case "ByteProperty":
		if legacy {
			return readString(reader)
		}
		first, err := readIdentifier(reader)
		if err != nil {
			return nil, err
		}

		if first == "None" {
			var value byte
			err := readAsType(reader, &value)
			return value, err
		}

		second, err := readIdentifier(reader)
		if err != nil {
			return nil, err
		}
		return first + "|" + second, nil
	case "BoolProperty":
		if legacy {
			var value [4]byte
			err := readAsType(reader, &value)
			return value[0] != 0, err
		}
		var value byte
		err := readAsType(reader, &value)
		return value != 0, err
	case "QWordProperty":
		var qword uint64
		err := readAsType(reader, &qword)
		return qword, err
	default:
		return nil, errors.New("unknown property type: " + typeName)
	}
}

func (parser *Parser) parseHeader(reader io.Reader, replay *rocket.Replay, headerInfo *HeaderData) error {
	var netVersion uint32

	if err := readAsType(reader, &headerInfo.EngineVersion); err != nil {
		return err
	}
	if err := readAsType(reader, &headerInfo.LicenseeVersion); err != nil {
		return err
	}

	if headerInfo.supportsNetVersion() {
		if err := readAsType(reader, &netVersion); err != nil {
			return err
		}
	}

	className, err := readString(reader)
	if err != nil {
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
	return info.EngineVersion > 865 && info.LicenseeVersion > 17
}

func readString(reader io.Reader) (string, error) {
	return readReplayString(reader, false)
}

func readIdentifier(reader io.Reader) (string, error) {
	return readReplayString(reader, true)
}

func readReplayString(reader io.Reader, identifier bool) (string, error) {
	stringLength, err := readInt32(reader)
	if err != nil {
		return "", err
	}
	if stringLength == 0 {
		return "", nil
	}

	size := int64(stringLength)
	if size < 0 {
		size *= -2
	}

	data, err := read(reader, size)
	if err != nil {
		return "", err
	}

	if stringLength < 0 {
		if data[len(data)-2] != 0 || data[len(data)-1] != 0 {
			return "", errors.New("unterminated UTF-16 string")
		}

		units := make([]uint16, (len(data)-2)/2)
		for i := range units {
			units[i] = binary.LittleEndian.Uint16(data[i*2:])
		}
		for i := 0; i < len(units); i++ {
			switch {
			case units[i] >= 0xd800 && units[i] <= 0xdbff:
				if i+1 >= len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
					return "", errors.New("invalid UTF-16 surrogate pair")
				}
				i++
			case units[i] >= 0xdc00 && units[i] <= 0xdfff:
				return "", errors.New("invalid UTF-16 surrogate pair")
			}
		}
		return string(utf16.Decode(units)), nil
	}

	if data[len(data)-1] != 0 {
		return "", errors.New("unterminated string")
	}
	data = data[:len(data)-1]
	if identifier {
		if !utf8.Valid(data) {
			return "", errors.New("invalid UTF-8 identifier")
		}
		return string(data), nil
	}

	return decodeWindows1252(data), nil
}

func decodeWindows1252(data []byte) string {
	var controls = [...]rune{
		0x20ac, 0x0081, 0x201a, 0x0192, 0x201e, 0x2026, 0x2020, 0x2021,
		0x02c6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008d, 0x017d, 0x008f,
		0x0090, 0x2018, 0x2019, 0x201c, 0x201d, 0x2022, 0x2013, 0x2014,
		0x02dc, 0x2122, 0x0161, 0x203a, 0x0153, 0x009d, 0x017e, 0x0178,
	}

	var result strings.Builder
	result.Grow(len(data))
	for _, value := range data {
		switch {
		case value < 0x80:
			result.WriteByte(value)
		case value < 0xa0:
			result.WriteRune(controls[value-0x80])
		default:
			result.WriteRune(rune(value))
		}
	}
	return result.String()
}

func read(reader io.Reader, numberOfBytes int64) ([]byte, error) {
	if numberOfBytes < 0 {
		return nil, fmt.Errorf("negative byte count: %d", numberOfBytes)
	}
	if numberOfBytes > int64(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("byte count too large: %d", numberOfBytes)
	}
	if remaining, ok := reader.(interface{ Len() int }); ok && int64(remaining.Len()) < numberOfBytes {
		return nil, io.ErrUnexpectedEOF
	}

	data, err := ioutil.ReadAll(io.LimitReader(reader, numberOfBytes))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != numberOfBytes {
		return nil, io.ErrUnexpectedEOF
	}

	return data, nil
}

func readAsType(reader io.Reader, interfaceType interface{}) error {
	return binary.Read(reader, binary.LittleEndian, interfaceType)
}

func readCount(reader io.Reader) (int32, error) {
	count, err := readInt32(reader)
	if err != nil {
		return 0, err
	}
	if count < 0 {
		return 0, fmt.Errorf("negative size or count: %d", count)
	}
	return count, nil
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
	listSize, err := readCount(reader)

	if err != nil {
		return nil, err
	}

	list := make([]string, 0)

	for index := int32(0); index < listSize; index++ {
		objectName, err := readString(reader)

		if err != nil {
			return nil, fmt.Errorf("string %d: %w", index, err)
		}

		list = append(list, objectName)
	}

	return list, nil
}

var replayCRCTable = func() [256]uint32 {
	var table [256]uint32
	for i := range table {
		crc := uint32(i) << 24
		for bit := 0; bit < 8; bit++ {
			if crc&0x80000000 != 0 {
				crc = crc<<1 ^ 0x04c11db7
			} else {
				crc <<= 1
			}
		}
		table[i] = crc
	}
	return table
}()

func replayCRC32(data []byte) uint32 {
	crc := uint32(0x10340dfe)
	for _, value := range data {
		crc = crc<<8 ^ replayCRCTable[byte(crc>>24)^value]
	}
	return crc ^ 0xffffffff
}

func validateCRC(section string, expected uint32, data []byte) error {
	actual := replayCRC32(data)
	if actual != expected {
		return fmt.Errorf("%s CRC mismatch: stored 0x%08x, computed 0x%08x", section, expected, actual)
	}
	return nil
}
