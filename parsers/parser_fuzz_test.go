//go:build go1.18
// +build go1.18

package parsers

import (
	"io/ioutil"
	"testing"

	"github.com/sbauer/go-rocket/source"
)

func FuzzParseReplay(f *testing.F) {
	for _, name := range []string{"07e9", "dropshot_rumble"} {
		data, err := ioutil.ReadFile("../samples/boxcars/" + name + ".replay")
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseWithOptions(source.FromBytes(data), ParseOptions{
			CRCValidation: CRCValidationNever,
			DecodeNetwork: true,
		})
	})
}
