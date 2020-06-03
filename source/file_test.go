package source

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFileSourceError_Should_Return_Error(t *testing.T) {
	badFileSource := &FileSource{error: errors.New("bad file")}

	assert.NotNil(t, badFileSource.Error())
}

func TestFileSource_Should_Return_Error_If_No_File(t *testing.T) {
	fileSource := FromFile("../samples/not.there")

	assert.NotNil(t, fileSource.Error())
}

/*

func Test_ReadBytes_Should_Return_Bytes(t *testing.T) {
	value := uint32(10)

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)

	reader := bytes.NewReader(data)

	result, _ := readNextBytes(bufio.NewReader(reader), 4)

	returnValue := binary.LittleEndian.Uint32(result)

	if value != returnValue {
		t.Errorf("Expected %d, got %d", value, returnValue)
	}
}

func Test_ReadNextString_Should_Read_And_Return_String(t *testing.T) {
	phrase := "Hello"
	phraseSlice := []byte(phrase)
	numberOfBytes := int32(len(phraseSlice))

	buffer := new(bytes.Buffer)

	binary.Write(buffer, binary.LittleEndian, numberOfBytes)
	binary.Write(buffer, binary.LittleEndian, []byte(phrase))

	reader := bytes.NewReader(buffer.Bytes())

	returnValue, _ := readNextString(bufio.NewReader(reader))

	if returnValue != phrase {
		t.Errorf("Expected %s, got %s", phrase, returnValue)
	}
}
*/
