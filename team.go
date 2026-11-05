package rocket

// Team contains a team's match result and players.
type Team struct {
	Name        string
	Index       int
	Score       int
	HasScore    bool
	IsWinner    bool
	Players     []string
	PlayerStats []PlayerStat
}

// NewTeam creates an empty team.
func NewTeam() *Team {
	return &Team{}
}
