package parsers

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

type mockSource struct {
	err    error
	reader io.Reader
}

func (s *mockSource) Reader() io.Reader { return s.reader }
func (s *mockSource) Error() error      { return s.err }

type freshReaderSource struct {
	data []byte
}

func (s *freshReaderSource) Reader() io.Reader { return bytes.NewReader(s.data) }
func (s *freshReaderSource) Error() error      { return nil }

func TestParseRejectsInvalidSource(t *testing.T) {
	sourceErr := errors.New("invalid source")
	tests := []struct {
		name string
		src  source.Source
		want error
	}{
		{name: "nil source"},
		{name: "source error", src: &mockSource{err: sourceErr}, want: sourceErr},
		{name: "nil reader", src: &mockSource{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.src)
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParseReplayFixtures(t *testing.T) {
	tests := []struct {
		name         string
		level        string
		teamSize     int32
		frames       int32
		goalCount    int
		keyframes    int
		networkSize  int
		networkStart []byte
		classes      int
		tickmark     string
		tickFrame    int32
	}{
		{"first", "EuroStad_OOB_Audio_Map", 2, 1477, 2, 6, 159744, []byte{0xa4, 0x77, 0xec, 0x41}, 39, "Team1Goal", 783},
		{"second", "Underwater_FX", 3, 6107, 4, 23, 959488, []byte{0x6c, 0xfb, 0xa8, 0x40}, 39, "Team0Goal", 244},
		{"third", "Underwater_FX", 3, 11314, 7, 42, 1583104, []byte{0x0f, 0xa8, 0xfd, 0x40}, 41, "Team1Goal", 752},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replay, err := Parse(source.FromFile("../samples/" + tt.name + ".replay"))
			if err != nil {
				t.Fatal(err)
			}
			if replay.HeaderInfo.ClassName != "TAGame.Replay_Soccar_TA" ||
				replay.HeaderInfo.EngineVersion != 868 ||
				replay.HeaderInfo.LicenseeVersion != 28 ||
				replay.HeaderInfo.NetVersion != 10 {
				t.Fatalf("unexpected header: %+v", replay.HeaderInfo)
			}
			if property := replay.Properties["TeamSize"]; property == nil || property.Value != tt.teamSize {
				t.Fatalf("unexpected TeamSize: %+v", property)
			}
			if property := replay.Properties["NumFrames"]; property == nil || property.Value != tt.frames {
				t.Fatalf("unexpected NumFrames: %+v", property)
			}
			if property := replay.Properties["Goals"]; property == nil || len(property.Groups) != tt.goalCount {
				t.Fatalf("unexpected goals: %+v", property)
			}
			if len(replay.Levels) == 0 || replay.Levels[0] != tt.level {
				t.Fatalf("unexpected levels: %v", replay.Levels)
			}
			if len(replay.Keyframes) != tt.keyframes {
				t.Fatalf("got %d keyframes, want %d", len(replay.Keyframes), tt.keyframes)
			}
			if len(replay.Tickmarks) != tt.goalCount || replay.Tickmarks[0].Description != tt.tickmark || replay.Tickmarks[0].Frame != tt.tickFrame {
				t.Fatalf("unexpected tickmarks: %+v", replay.Tickmarks)
			}
			if len(replay.NetworkData) != tt.networkSize || !bytes.HasPrefix(replay.NetworkData, tt.networkStart) {
				t.Fatalf("unexpected network data: %d bytes", len(replay.NetworkData))
			}
			if len(replay.ClassIndices) != tt.classes || len(replay.NetCache) != 34 {
				t.Fatalf("got %d class indices and %d cache entries", len(replay.ClassIndices), len(replay.NetCache))
			}
			for _, class := range replay.ClassIndices {
				if class.Index < 0 || int(class.Index) >= len(replay.Objects) || replay.Objects[class.Index] != class.Class {
					t.Fatalf("invalid class mapping: %+v", class)
				}
			}
			var foundMapping bool
			for _, entry := range replay.NetCache {
				if entry.ObjectIndex < 0 || int(entry.ObjectIndex) >= len(replay.Objects) {
					t.Fatalf("invalid cache object: %+v", entry)
				}
				for _, property := range entry.Properties {
					if property.ObjectIndex < 0 || int(property.ObjectIndex) >= len(replay.Objects) {
						t.Fatalf("invalid cache property: %+v", property)
					}
				}
				if entry.ObjectIndex == 21 && entry.ParentID == 0 && entry.CacheID == 20 &&
					len(entry.Properties) > 1 && entry.Properties[0].ObjectIndex == 1 && entry.Properties[0].StreamID == 0 &&
					entry.Properties[1].ObjectIndex == 2 && entry.Properties[1].StreamID == 1 {
					foundMapping = true
				}
			}
			if !foundMapping {
				t.Fatal("expected the known network cache mapping")
			}
		})
	}
}

func TestParseUsesOneSourceReader(t *testing.T) {
	data, err := ioutil.ReadFile("../samples/first.replay")
	if err != nil {
		t.Fatal(err)
	}
	replay, err := Parse(&freshReaderSource{data: data})
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.ClassIndices) != 39 {
		t.Fatalf("got %d class indices", len(replay.ClassIndices))
	}
}

func TestParseRejectsTruncatedReplay(t *testing.T) {
	data, err := ioutil.ReadFile("../samples/first.replay")
	if err != nil {
		t.Fatal(err)
	}
	headerEnd := 8 + int(binary.LittleEndian.Uint32(data))
	for _, length := range []int{0, 3, 4, 8, 12, headerEnd - 1, headerEnd, headerEnd + 4, len(data) - 1} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			_, err := Parse(source.FromBytes(data[:length]))
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("got %v, want a truncation error", err)
			}
		})
	}
}

func TestParseRejectsNegativeSectionSizes(t *testing.T) {
	data, err := ioutil.ReadFile("../samples/first.replay")
	if err != nil {
		t.Fatal(err)
	}
	bodyStart := 8 + int(binary.LittleEndian.Uint32(data))
	for _, offset := range []int{0, bodyStart} {
		mutated := append([]byte(nil), data...)
		binary.LittleEndian.PutUint32(mutated[offset:], ^uint32(0))
		if _, err := Parse(source.FromBytes(mutated)); err == nil {
			t.Fatalf("expected an error for negative section size at offset %d", offset)
		}
	}
}

func TestAdditionalFixturesRejectMalformedData(t *testing.T) {
	for _, sample := range []struct {
		name        string
		classOffset int
	}{
		{"07e9", 16},
		{"MaxNumJumps", 20},
	} {
		t.Run(sample.name, func(t *testing.T) {
			data, err := ioutil.ReadFile("../samples/boxcars/" + sample.name + ".replay")
			if err != nil {
				t.Fatal(err)
			}
			bodyStart := 8 + int(binary.LittleEndian.Uint32(data))
			check := func(name string, input []byte, want string, truncation bool) {
				t.Helper()
				t.Run(name, func(t *testing.T) {
					_, err := ParseWithOptions(source.FromBytes(input), ParseOptions{
						CRCValidation: CRCValidationNever, DecodeNetwork: true,
					})
					if err == nil || !strings.Contains(err.Error(), want) {
						t.Fatalf("got %v, want an error containing %q", err, want)
					}
					if truncation && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
						t.Fatalf("got %v, want a truncation error", err)
					}
				})
			}
			check("truncated header", data[:bodyStart-1], "header", true)
			check("truncated body", data[:len(data)-1], "body", true)
			for _, tt := range []struct {
				name   string
				offset int
				value  uint32
				want   string
			}{
				{"oversized string", sample.classOffset, 0x7fffffff, "header"},
				{"oversized UTF-16 string", sample.classOffset, 0x80000000, "header"},
				{"negative level count", bodyStart + 8, 0xffffffff, "levels"},
				{"oversized level count", bodyStart + 8, 0x7fffffff, "levels"},
			} {
				mutated := append([]byte(nil), data...)
				binary.LittleEndian.PutUint32(mutated[tt.offset:], tt.value)
				check(tt.name, mutated, tt.want, false)
			}
		})
	}
}

func TestParseValidatesSectionCRCs(t *testing.T) {
	data, err := ioutil.ReadFile("../samples/first.replay")
	if err != nil {
		t.Fatal(err)
	}
	headerSize := int(binary.LittleEndian.Uint32(data))
	bodyStart := 8 + headerSize
	bodySize := int(binary.LittleEndian.Uint32(data[bodyStart:]))
	tests := []struct {
		name        string
		offset      int
		wantSection string
	}{
		{name: "header", offset: 8, wantSection: "header"},
		{name: "body", offset: bodyStart + 8 + bodySize - 1, wantSection: "body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutated := append([]byte(nil), data...)
			mutated[tt.offset] ^= 1
			_, err := Parse(source.FromBytes(mutated))
			if err == nil || !strings.Contains(err.Error(), tt.wantSection+" CRC mismatch") {
				t.Fatalf("got %v, want %s CRC mismatch", err, tt.wantSection)
			}
		})
	}
}

func TestParseCRCValidationOptions(t *testing.T) {
	data, err := ioutil.ReadFile("../samples/first.replay")
	if err != nil {
		t.Fatal(err)
	}
	headerSize := int(binary.LittleEndian.Uint32(data))
	bodyStart := 8 + headerSize
	bodySize := int(binary.LittleEndian.Uint32(data[bodyStart:]))
	validButCorrupt := append([]byte(nil), data...)
	validButCorrupt[8] ^= 1
	validButCorrupt[bodyStart+8+bodySize-1] ^= 1
	for _, mode := range []CRCValidationMode{CRCValidationNever, CRCValidationOnParseError} {
		if _, err := ParseWithOptions(source.FromBytes(validButCorrupt), ParseOptions{CRCValidation: mode}); err != nil {
			t.Fatalf("mode %d rejected parseable data with CRC validation disabled: %v", mode, err)
		}
	}

	headerParseFailure := append([]byte(nil), data...)
	headerParseFailure[8+headerSize-1] ^= 1
	if _, err := ParseWithOptions(source.FromBytes(headerParseFailure), ParseOptions{CRCValidation: CRCValidationOnParseError}); err == nil ||
		!strings.Contains(err.Error(), "header CRC mismatch") {
		t.Fatalf("got %v, want header CRC mismatch after a header parse failure", err)
	}

	bodyParseFailure := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(bodyParseFailure[bodyStart+8:], ^uint32(0))
	if _, err := ParseWithOptions(source.FromBytes(bodyParseFailure), ParseOptions{CRCValidation: CRCValidationOnParseError}); err == nil ||
		!strings.Contains(err.Error(), "body CRC mismatch") {
		t.Fatalf("got %v, want body CRC mismatch after a body parse failure", err)
	}
}

func TestReadReplayString(t *testing.T) {
	tests := []struct {
		name    string
		length  int32
		data    []byte
		want    string
		wantErr bool
	}{
		{name: "empty", length: 0, want: ""},
		{name: "ascii", length: 6, data: []byte("hello\x00"), want: "hello"},
		{name: "windows-1252", length: 3, data: []byte{0x80, 0xe9, 0}, want: "\u20ac\u00e9"},
		{name: "utf16", length: -4, data: []byte{0x41, 0, 0x3d, 0xd8, 0x80, 0xde, 0, 0}, want: "A\U0001f680"},
		{name: "empty utf16", length: -1, data: []byte{0, 0}, want: ""},
		{name: "truncated", length: -3, data: []byte{0x41, 0}, wantErr: true},
		{name: "huge length", length: -2147483648, wantErr: true},
		{name: "unterminated", length: 1, data: []byte{'x'}, wantErr: true},
		{name: "unterminated utf16", length: -1, data: []byte{'x', 0}, wantErr: true},
		{name: "invalid surrogate", length: -2, data: []byte{0x3d, 0xd8, 0, 0}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data bytes.Buffer
			writeValue(t, &data, tt.length)
			data.Write(tt.data)
			got, err := readString(&data)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got %q, want an error", got)
				}
			} else if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}

	if _, err := readString(bytes.NewReader([]byte{1, 0})); err == nil {
		t.Fatal("expected an error for a truncated string length")
	}
}

func TestReadIdentifierUsesUTF8(t *testing.T) {
	var data bytes.Buffer
	writeString(t, &data, "\u00e9")
	got, err := readIdentifier(&data)
	if err != nil || got != "\u00e9" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestPropertyNameContainingNone(t *testing.T) {
	var data bytes.Buffer
	writeString(t, &data, "NoneOfYourBusiness")
	writeString(t, &data, "IntProperty")
	writeValue(t, &data, uint32(4))
	writeValue(t, &data, uint32(0))
	writeValue(t, &data, int32(42))
	writeString(t, &data, "None")

	properties, err := (&Parser{}).parseProperties(&data, nil)
	if err != nil {
		t.Fatal(err)
	}
	property := properties["NoneOfYourBusiness"]
	if property == nil || property.Value != int32(42) {
		t.Fatalf("unexpected property: %+v", property)
	}
}

func TestPropertyReportsTruncatedData(t *testing.T) {
	for _, kind := range []string{"IntProperty", "FloatProperty", "BoolProperty", "QWordProperty"} {
		t.Run(kind, func(t *testing.T) {
			_, err := (&Parser{}).parsePropertyValue(bytes.NewReader(nil), kind)
			if err == nil {
				t.Fatal("expected an error")
			}
		})
	}

	var data bytes.Buffer
	writeString(t, &data, "Score")
	writeString(t, &data, "IntProperty")
	writeValue(t, &data, uint32(4))
	if _, err := (&Parser{}).parseProperty(&data, nil); err == nil {
		t.Fatal("expected an error for the missing property index")
	}
}

func TestPropertyGroupReportsFirstError(t *testing.T) {
	var data bytes.Buffer
	writeValue(t, &data, int32(2))
	writeString(t, &data, "Invalid")
	writeString(t, &data, "UnsupportedProperty")
	writeValue(t, &data, uint32(0))
	writeValue(t, &data, uint32(0))
	writeString(t, &data, "None")

	if _, err := (&Parser{}).parsePropertyGroups(&data, nil); err == nil {
		t.Fatal("expected an error from the first group")
	}
}

func TestNestedStructProperties(t *testing.T) {
	var data bytes.Buffer
	writeProperty := func(name, kind string) {
		writeString(t, &data, name)
		writeString(t, &data, kind)
		writeValue(t, &data, uint32(0))
		writeValue(t, &data, uint32(0))
	}
	writeProperty("PlayerID", "StructProperty")
	writeString(t, &data, "UniqueNetId")
	writeProperty("Uid", "QWordProperty")
	writeValue(t, &data, int64(123))
	writeProperty("Nested", "StructProperty")
	writeString(t, &data, "Details")
	writeProperty("Score", "IntProperty")
	writeValue(t, &data, int32(7))
	writeString(t, &data, "None")
	writeString(t, &data, "None")
	writeString(t, &data, "None")

	properties, err := (&Parser{}).parseProperties(&data, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, ok := properties["PlayerID"].Value.(rocket.StructProperty)
	if !ok || id.Name != "UniqueNetId" || id.Fields["Uid"].Value != int64(123) {
		t.Fatalf("unexpected player ID: %+v", properties["PlayerID"])
	}
	nested, ok := id.Fields["Nested"].Value.(rocket.StructProperty)
	if !ok || nested.Name != "Details" || nested.Fields["Score"].Value != int32(7) {
		t.Fatalf("unexpected nested properties: %+v", id.Fields["Nested"])
	}
}

func TestPropertyNestingLimit(t *testing.T) {
	nested := func(depth int) *bytes.Buffer {
		var data bytes.Buffer
		for i := 0; i < depth; i++ {
			writeString(t, &data, "Nested")
			writeString(t, &data, "StructProperty")
			writeValue(t, &data, uint32(0))
			writeValue(t, &data, uint32(0))
			writeString(t, &data, "NestedType")
		}
		for i := 0; i <= depth; i++ {
			writeString(t, &data, "None")
		}
		return &data
	}
	if _, err := (&Parser{}).parseProperties(nested(maxPropertyDepth-1), nil); err != nil {
		t.Fatalf("maximum supported nesting: %v", err)
	}
	_, err := (&Parser{}).parseProperties(nested(maxPropertyDepth), nil)
	if err == nil || !strings.Contains(err.Error(), "property nesting exceeds") {
		t.Fatalf("got %v, want a nesting error", err)
	}
}

func TestNestedStructRejectsTruncation(t *testing.T) {
	var data bytes.Buffer
	writeString(t, &data, "PlayerID")
	writeString(t, &data, "StructProperty")
	writeValue(t, &data, uint32(0))
	writeValue(t, &data, uint32(0))
	missingName := append([]byte(nil), data.Bytes()...)
	writeString(t, &data, "UniqueNetId")
	missingFields := append([]byte(nil), data.Bytes()...)
	writeString(t, &data, "Uid")
	writeString(t, &data, "QWordProperty")
	writeValue(t, &data, uint32(8))
	writeValue(t, &data, uint32(0))
	writeValue(t, &data, uint32(1))

	for name, input := range map[string][]byte{
		"missing struct name": missingName,
		"missing fields":      missingFields,
		"truncated field":     data.Bytes(),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := (&Parser{}).parseProperties(bytes.NewReader(input), nil); err == nil {
				t.Fatal("expected a truncation error")
			}
		})
	}
}

func TestBytePropertyWithoutEnum(t *testing.T) {
	var data bytes.Buffer
	writeString(t, &data, "None")
	data.WriteByte(7)
	value, err := (&Parser{}).parsePropertyValue(&data, "ByteProperty")
	if err != nil || value != byte(7) {
		t.Fatalf("got %v, %v; want byte value 7", value, err)
	}
}

func TestParseNetVersionBoundary(t *testing.T) {
	tests := []struct {
		major, minor uint32
		netVersion   uint32
		hasNet       bool
	}{
		{865, 18, 0, false},
		{866, 18, 7, true},
		{868, 17, 0, false},
		{868, 18, 10, true},
	}
	for _, tt := range tests {
		var header bytes.Buffer
		writeValue(t, &header, tt.major)
		writeValue(t, &header, tt.minor)
		if tt.hasNet {
			writeValue(t, &header, tt.netVersion)
		}
		writeString(t, &header, "Replay")
		writeString(t, &header, "None")

		var replay bytes.Buffer
		writeValue(t, &replay, int32(header.Len()))
		writeValue(t, &replay, replayCRC32(header.Bytes()))
		replay.Write(header.Bytes())
		replay.Write(emptyBody(t))

		parsed, err := Parse(source.FromBytes(replay.Bytes()))
		if err != nil {
			t.Fatalf("version %d/%d: %v", tt.major, tt.minor, err)
		}
		if parsed.HeaderInfo.NetVersion != int(tt.netVersion) {
			t.Fatalf("version %d/%d: got net version %d", tt.major, tt.minor, parsed.HeaderInfo.NetVersion)
		}
	}
}

func TestBodyParserRejectsNegativeCounts(t *testing.T) {
	sections := []string{"levels", "keyframes", "network data", "debug data", "tickmarks", "packages", "objects", "names", "class indices", "network cache"}
	for index, section := range sections {
		t.Run(section, func(t *testing.T) {
			body := emptyBody(t)
			binary.LittleEndian.PutUint32(body[8+index*4:], ^uint32(0))
			setBodyCRC(body)
			_, err := NewBodyParser(source.FromBytes(body)).Parse()
			if err == nil || !strings.Contains(err.Error(), section) {
				t.Fatalf("got %v, want an error in %s", err, section)
			}
		})
	}
}

func TestBodyParserReportsTruncatedEntries(t *testing.T) {
	tests := []struct {
		section string
		values  []int32
	}{
		{"debug data", []int32{0, 0, 0, 1, 42}},
		{"tickmarks", []int32{0, 0, 0, 0, 1, 0}},
		{"class indices", []int32{0, 0, 0, 0, 0, 0, 0, 0, 1, 0}},
		{"network cache", []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 2, 1, 5}},
	}
	for _, tt := range tests {
		t.Run(tt.section, func(t *testing.T) {
			var data bytes.Buffer
			writeValue(t, &data, int32(4*len(tt.values)))
			writeValue(t, &data, uint32(0))
			for _, value := range tt.values {
				writeValue(t, &data, value)
			}
			body := data.Bytes()
			setBodyCRC(body)
			_, err := NewBodyParser(source.FromBytes(body)).Parse()
			if err == nil || !strings.Contains(err.Error(), tt.section) {
				t.Fatalf("got %v, want a truncation error in %s", err, tt.section)
			}
		})
	}
}

func writeValue(t *testing.T, data *bytes.Buffer, value interface{}) {
	t.Helper()
	if err := binary.Write(data, binary.LittleEndian, value); err != nil {
		t.Fatal(err)
	}
}

func writeString(t *testing.T, data *bytes.Buffer, value string) {
	t.Helper()
	writeValue(t, data, int32(len(value)+1))
	data.WriteString(value)
	data.WriteByte(0)
}

func emptyBody(t *testing.T) []byte {
	t.Helper()
	var body bytes.Buffer
	writeValue(t, &body, int32(40))
	writeValue(t, &body, uint32(0))
	for i := 0; i < 10; i++ {
		writeValue(t, &body, int32(0))
	}
	setBodyCRC(body.Bytes())
	return body.Bytes()
}

func setBodyCRC(body []byte) {
	binary.LittleEndian.PutUint32(body[4:], replayCRC32(body[8:]))
}
