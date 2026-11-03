package parsers

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

type BodyParser struct {
	reader        io.Reader
	dataSource    source.Source
	crcValidation CRCValidationMode
	crc           uint32
	data          []byte
}

type BodyParserResult struct {
	levels       []string
	keyframes    []*rocket.Keyframe
	networkData  []byte
	debugData    []*rocket.DebugData
	tickmarks    []*rocket.Tickmark
	packages     []string
	objects      []string
	names        []string
	classIndices []*rocket.ClassIndex
	netCache     []*rocket.ClassNetCache
}

func NewBodyParser(dataSource source.Source) *BodyParser {
	if dataSource == nil {
		return nil
	}

	return &BodyParser{dataSource: dataSource}
}

func (parser *BodyParser) Parse() (*BodyParserResult, error) {
	if parser == nil || parser.dataSource == nil {
		return nil, errors.New("source is nil")
	}

	if err := parser.dataSource.Error(); err != nil {
		return nil, err
	}

	reader := parser.dataSource.Reader()
	if reader == nil {
		return nil, errors.New("source reader is nil")
	}
	return parser.parse(reader)
}

func (parser *BodyParser) parse(reader io.Reader) (result *BodyParserResult, parseErr error) {
	if err := parser.initializeBuffer(reader); err != nil {
		return nil, err
	}
	if parser.crcValidation == CRCValidationAlways {
		if err := validateCRC("body", parser.crc, parser.data); err != nil {
			return nil, err
		}
	}
	defer func() {
		if parseErr != nil && parser.crcValidation == CRCValidationOnParseError {
			if err := validateCRC("body", parser.crc, parser.data); err != nil {
				parseErr = fmt.Errorf("%w (body parsing failed: %v)", err, parseErr)
			}
		}
	}()

	levels, err := parseStringList(parser.reader)
	if err != nil {
		return nil, fmt.Errorf("levels: %w", err)
	}

	keyframes, err := parser.parseKeyFrames()
	if err != nil {
		return nil, fmt.Errorf("keyframes: %w", err)
	}

	networkBuffer, err := parser.parseNetworkData()
	if err != nil {
		return nil, fmt.Errorf("network data: %w", err)
	}

	debugInfo, err := parser.parseDebugInfo()
	if err != nil {
		return nil, fmt.Errorf("debug data: %w", err)
	}

	tickmarks, err := parser.parseTickmarks()
	if err != nil {
		return nil, fmt.Errorf("tickmarks: %w", err)
	}

	packages, err := parseStringList(parser.reader)
	if err != nil {
		return nil, fmt.Errorf("packages: %w", err)
	}

	objects, err := parseStringList(parser.reader)
	if err != nil {
		return nil, fmt.Errorf("objects: %w", err)
	}

	names, err := parseStringList(parser.reader)
	if err != nil {
		return nil, fmt.Errorf("names: %w", err)
	}

	classIndices, err := parser.parseClassIndices()
	if err != nil {
		return nil, fmt.Errorf("class indices: %w", err)
	}

	netCache, err := parser.parseNetCache()
	if err != nil {
		return nil, fmt.Errorf("network cache: %w", err)
	}

	result = &BodyParserResult{
		levels:       levels,
		keyframes:    keyframes,
		networkData:  networkBuffer,
		debugData:    debugInfo,
		tickmarks:    tickmarks,
		packages:     packages,
		objects:      objects,
		names:        names,
		classIndices: classIndices,
		netCache:     netCache,
	}

	return result, nil
}

func (bp *BodyParser) parseTickmarks() ([]*rocket.Tickmark, error) {
	tickmarkSize, err := readCount(bp.reader)

	if err != nil {
		return nil, err
	}

	tickmarks := make([]*rocket.Tickmark, 0)

	for index := int32(0); index < tickmarkSize; index++ {
		description, err := readString(bp.reader)
		if err != nil {
			return nil, fmt.Errorf("tickmark %d description: %w", index, err)
		}

		frame, err := readInt32(bp.reader)
		if err != nil {
			return nil, fmt.Errorf("tickmark %d frame: %w", index, err)
		}

		tickmarks = append(tickmarks, &rocket.Tickmark{Frame: frame, Description: description})
	}

	return tickmarks, nil
}

func (bp *BodyParser) parseDebugInfo() ([]*rocket.DebugData, error) {
	debugInfoSize, err := readCount(bp.reader)

	if err != nil {
		return nil, err
	}

	debugData := make([]*rocket.DebugData, 0)

	for index := int32(0); index < debugInfoSize; index++ {
		frame, err := readInt32(bp.reader)
		if err != nil {
			return nil, fmt.Errorf("debug data %d frame: %w", index, err)
		}

		user, err := readString(bp.reader)
		if err != nil {
			return nil, fmt.Errorf("debug data %d user: %w", index, err)
		}

		text, err := readString(bp.reader)
		if err != nil {
			return nil, fmt.Errorf("debug data %d text: %w", index, err)
		}

		debugData = append(debugData, &rocket.DebugData{User: user, Frame: frame, Text: text})
	}

	return debugData, nil
}

func (bodyParser *BodyParser) parseNetworkData() ([]byte, error) {
	networkSize, err := readCount(bodyParser.reader)
	if err != nil {
		return nil, err
	}

	return read(bodyParser.reader, int64(networkSize))
}

func (bodyParser *BodyParser) parseKeyFrames() ([]*rocket.Keyframe, error) {
	keyframeSize, err := readCount(bodyParser.reader)

	if err != nil {
		return nil, err
	}

	keyframes := make([]*rocket.Keyframe, 0)

	for index := int32(0); index < keyframeSize; index++ {
		time, err := readFloat32(bodyParser.reader)
		if err != nil {
			return nil, fmt.Errorf("keyframe %d time: %w", index, err)
		}

		frame, err := readInt32(bodyParser.reader)
		if err != nil {
			return nil, fmt.Errorf("keyframe %d frame: %w", index, err)
		}

		position, err := readInt32(bodyParser.reader)
		if err != nil {
			return nil, fmt.Errorf("keyframe %d position: %w", index, err)
		}

		keyframes = append(keyframes, &rocket.Keyframe{Time: time, Frame: frame, Position: position})
	}

	return keyframes, nil
}

func (bodyParser *BodyParser) parseClassIndices() ([]*rocket.ClassIndex, error) {
	count, err := readCount(bodyParser.reader)
	if err != nil {
		return nil, err
	}

	indices := make([]*rocket.ClassIndex, 0)
	for i := int32(0); i < count; i++ {
		class, err := readIdentifier(bodyParser.reader)
		if err != nil {
			return nil, fmt.Errorf("class index %d name: %w", i, err)
		}
		index, err := readInt32(bodyParser.reader)
		if err != nil {
			return nil, fmt.Errorf("class index %d index: %w", i, err)
		}
		indices = append(indices, &rocket.ClassIndex{Class: class, Index: index})
	}
	return indices, nil
}

func (bodyParser *BodyParser) parseNetCache() ([]*rocket.ClassNetCache, error) {
	count, err := readCount(bodyParser.reader)
	if err != nil {
		return nil, err
	}

	cache := make([]*rocket.ClassNetCache, 0)
	for i := int32(0); i < count; i++ {
		var header struct {
			ObjectIndex int32
			ParentID    int32
			CacheID     int32
		}
		if err := readAsType(bodyParser.reader, &header); err != nil {
			return nil, fmt.Errorf("cache %d: %w", i, err)
		}

		propertyCount, err := readCount(bodyParser.reader)
		if err != nil {
			return nil, fmt.Errorf("cache %d property count: %w", i, err)
		}
		entry := &rocket.ClassNetCache{
			ObjectIndex: header.ObjectIndex,
			ParentID:    header.ParentID,
			CacheID:     header.CacheID,
			Properties:  make([]*rocket.NetCacheProperty, 0),
		}
		for j := int32(0); j < propertyCount; j++ {
			property := &rocket.NetCacheProperty{}
			if err := readAsType(bodyParser.reader, property); err != nil {
				return nil, fmt.Errorf("cache %d property %d: %w", i, j, err)
			}
			entry.Properties = append(entry.Properties, property)
		}
		cache = append(cache, entry)
	}
	return cache, nil
}

func (bodyParser *BodyParser) initializeBuffer(reader io.Reader) error {
	size, err := readCount(reader)
	if err != nil {
		return fmt.Errorf("body size: %w", err)
	}
	bodyParser.crc, err = readUInt32(reader)
	if err != nil {
		return fmt.Errorf("body CRC: %w", err)
	}
	data, err := read(reader, int64(size))
	if err != nil {
		return fmt.Errorf("body data: %w", err)
	}
	bodyParser.data = data
	bodyParser.reader = bytes.NewReader(data)
	return nil
}
