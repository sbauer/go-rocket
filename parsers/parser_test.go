package parsers

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/sbauer/go-rocket/source"
)

var invalidSource source.Source

var validReplayFile string

func init() {
	invalidSource = &mockSource{error: errors.New("invalid file")}
	validReplayFile = "samples\\first.replay"
}

type mockSource struct {
	error error
}

func (s *mockSource) Reader() io.Reader {
	return nil
}

func (s *mockSource) Error() error {
	return s.error
}

func TestReplayParser_Parse_Should_Return_Error_For_Nil_Source(t *testing.T) {
	_, err := Parse(invalidSource)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func Test_ParserCanReadFile(t *testing.T) {
	results, _ := Parse(source.FromFile("../samples/first.replay"))

	for i := 0; i < len(results.Names); i++ {
		fmt.Println(i, ". ", results.Names[i])
	}
}
