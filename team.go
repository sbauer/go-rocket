package rocket

// Team is not currently used
type Team struct {
	Name     string
	Score    int
	IsWinner bool
	Players  []string
}

// NewTeam is not currently used
func NewTeam() *Team {
	return &Team{}
}
