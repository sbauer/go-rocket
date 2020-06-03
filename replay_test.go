package rocket

import (
	"testing"
)

func TestNewReplay(t *testing.T) {
	replay := NewReplay()
	if replay == nil {
		t.Errorf("NewReplay expected replay, got nil")
	}
}
