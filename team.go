package rocket

// Team contains a team's match result and players.
type Team struct {
	Name          string
	Index         int
	Score         int
	HasScore      bool
	ScoreDerived  bool // True when inferred from the replay's Goals property.
	IsWinner      bool
	WinnerDerived bool // True when inferred from team scores rather than an explicit winner property.
	Players       []string
	PlayerStats   []PlayerStat
}

// NewTeam creates an empty team.
func NewTeam() *Team {
	return &Team{}
}
