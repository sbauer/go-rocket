package source

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFromBytes_Should_Return_A_Source_With_Error_If_Nil_Data(t *testing.T) {
	source := FromBytes(nil)

	assert.NotNil(t, source.Error())
}

func TestFromBytes_Should_Return_Source_With_Reader(t *testing.T) {
	source := FromBytes(make([]byte, 4))

	assert.NotNil(t, source.Reader())
}
