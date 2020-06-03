package rocket

import "testing"

func TestNewTeamShouldReturnTeam(t *testing.T) {
	team := NewTeam()

	if team == nil {
		t.Errorf("NewTeam expected team, got nil")
	}
}
