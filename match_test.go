package rocket_test

import (
	"strings"
	"testing"

	rocket "github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/parsers"
	"github.com/sbauer/go-rocket/source"
)

func TestReplayMatchData(t *testing.T) {
	tests := []struct {
		name                                       string
		mapName, match, date                       string
		teamSize                                   int32
		goals, highlights                          int
		players                                    int
		team0Score, team1Score                     *int32
		winningTeam                                *int32
		team0Derived, team1Derived, winningDerived bool
	}{
		{
			name: "first", mapName: "EuroStadium_P", match: "Online", date: "2020-03-17 13-44-05",
			teamSize: 2, goals: 2, highlights: 2, players: 4,
			team0Score: int32Pointer(0), team1Score: int32Pointer(2), winningTeam: int32Pointer(1),
			team0Derived: true, winningDerived: true,
		},
		{
			name: "second", mapName: "Underwater_P", match: "Online", date: "2020-03-19 22-46-56",
			teamSize: 3, goals: 4, highlights: 4, players: 6,
			team0Score: int32Pointer(4), team1Score: int32Pointer(0), winningTeam: int32Pointer(0),
			team1Derived: true, winningDerived: true,
		},
		{
			name: "third", mapName: "Underwater_P", match: "Online", date: "2020-03-21 17-23-05",
			teamSize: 3, goals: 7, highlights: 7, players: 6,
			team0Score: int32Pointer(4), team1Score: int32Pointer(3), winningTeam: int32Pointer(0),
			winningDerived: true,
		},
		{
			name: "boxcars/MaxNumJumps", mapName: "UtopiaStadium_P", match: "Offline", date: "2026-10-11 15-20-50",
			teamSize: 1, goals: 7, highlights: 13, players: 2,
			team0Score: int32Pointer(0), team1Score: int32Pointer(7), winningTeam: int32Pointer(1),
			team0Derived: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replay, err := parsers.Parse(source.FromFile("samples/" + tt.name + ".replay"))
			if err != nil {
				t.Fatal(err)
			}
			match, err := replay.MatchData()
			if err != nil {
				t.Fatal(err)
			}
			if match.MapName != tt.mapName || match.MatchType != tt.match || match.Date != tt.date ||
				match.TeamSize != tt.teamSize || len(match.Goals) != tt.goals ||
				len(match.Highlights) != tt.highlights || len(match.PlayerStats) != tt.players {
				t.Fatalf("unexpected match summary: %+v", match)
			}
			assertOptionalInt32(t, "Team0Score", match.Team0Score, tt.team0Score)
			assertOptionalInt32(t, "Team1Score", match.Team1Score, tt.team1Score)
			assertOptionalInt32(t, "WinningTeam", match.WinningTeam, tt.winningTeam)
			if match.Team0ScoreDerived != tt.team0Derived || match.Team1ScoreDerived != tt.team1Derived ||
				match.WinningTeamDerived != tt.winningDerived {
				t.Errorf("derived fields: scores=%v/%v winner=%v; want %v/%v/%v",
					match.Team0ScoreDerived, match.Team1ScoreDerived, match.WinningTeamDerived,
					tt.team0Derived, tt.team1Derived, tt.winningDerived)
			}
			if len(match.Teams) != 2 || match.Teams[0].Index != 0 || match.Teams[1].Index != 1 {
				t.Fatalf("unexpected teams: %+v", match.Teams)
			}
			for i, score := range []*int32{match.Team0Score, match.Team1Score} {
				if match.Teams[i].HasScore != (score != nil) {
					t.Errorf("team %d HasScore=%v, score=%v", i, match.Teams[i].HasScore, score)
				}
				if score != nil && match.Teams[i].Score != int(*score) {
					t.Errorf("team %d score=%d, want %d", i, match.Teams[i].Score, *score)
				}
			}
			if match.Teams[0].ScoreDerived != tt.team0Derived || match.Teams[1].ScoreDerived != tt.team1Derived {
				t.Errorf("team score derivation flags=%v/%v, want %v/%v",
					match.Teams[0].ScoreDerived, match.Teams[1].ScoreDerived,
					tt.team0Derived, tt.team1Derived)
			}
			for i, team := range match.Teams {
				wantWinnerDerived := tt.winningDerived && tt.winningTeam != nil && *tt.winningTeam == int32(i)
				if team.WinnerDerived != wantWinnerDerived {
					t.Errorf("team %d WinnerDerived=%v, want %v", i, team.WinnerDerived, wantWinnerDerived)
				}
			}
			if len(match.Teams[0].Players)+len(match.Teams[1].Players) != tt.players {
				t.Fatalf("team player counts do not match player stats: %+v", match.Teams)
			}
			if len(match.Goals) != 0 {
				if match.Goals[0].Frame < 0 || match.Goals[0].PlayerName == "" ||
					(match.Goals[0].PlayerTeam != 0 && match.Goals[0].PlayerTeam != 1) {
					t.Errorf("unexpected first goal: %+v", match.Goals[0])
				}
			}
			if len(match.Highlights) != 0 && match.Highlights[0].Frame < 0 {
				t.Errorf("unexpected first highlight: %+v", match.Highlights[0])
			}
			if len(match.PlayerStats) != 0 {
				player := match.PlayerStats[0]
				if player.Name == "" || player.Platform == "" || player.Team == nil ||
					player.Score < 0 || player.Goals < 0 || player.Assists < 0 ||
					player.Saves < 0 || player.Shots < 0 {
					t.Errorf("unexpected first player stat: %+v", player)
				}
				team := match.Teams[*player.Team]
				if team.PlayerStats[0].Name != player.Name {
					t.Errorf("player stats not grouped with team: %+v", team)
				}
				if tt.name == "first" && (player.Score != 242 || player.Goals != 2 ||
					player.Assists != 0 || player.Saves != 0 || player.Shots != 3 ||
					player.IsBot || player.OnlineID == nil || player.Platform != "OnlinePlatform|OnlinePlatform_Steam") {
					t.Errorf("unexpected typed player stats: %+v", player)
				}
			}
			if len(match.Goals) > 0 && tt.name == "first" && match.Goals[0].Frame != 828 {
				t.Errorf("first goal frame=%d, want 828", match.Goals[0].Frame)
			}
			if len(match.Highlights) > 0 && tt.name == "first" &&
				(match.Highlights[0].Frame != 828 || match.Highlights[0].GoalActorName != "") {
				t.Errorf("unexpected first highlight: %+v", match.Highlights[0])
			}
			if match.WinningTeam != nil {
				if !match.Teams[*match.WinningTeam].IsWinner {
					t.Errorf("winning team %d not marked winner", *match.WinningTeam)
				}
			} else if match.Team0Score != nil && match.Team1Score != nil {
				wantWinner := int32(-1)
				if *match.Team0Score > *match.Team1Score {
					wantWinner = 0
				} else if *match.Team1Score > *match.Team0Score {
					wantWinner = 1
				}
				for i, team := range match.Teams {
					if team.IsWinner != (int32(i) == wantWinner) {
						t.Errorf("team %d winner=%v, expected winner index %d", i, team.IsWinner, wantWinner)
					}
				}
			}
		})
	}
}

func TestMatchDataDerivedResults(t *testing.T) {
	tests := []struct {
		name        string
		properties  map[string]*rocket.Property
		score0      *int32
		score1      *int32
		winner      *int32
		score0Deriv bool
		score1Deriv bool
		winDeriv    bool
	}{
		{
			name: "derive zero score and winner from complete goals",
			properties: map[string]*rocket.Property{
				"Goals":      matchGoalsProperty(1, 1),
				"Team1Score": {Type: "IntProperty", Value: int32(2)},
			},
			score0: int32Pointer(0), score1: int32Pointer(2), winner: int32Pointer(1),
			score0Deriv: true, winDeriv: true,
		},
		{
			name: "tie has no winner",
			properties: map[string]*rocket.Property{
				"Goals": matchGoalsProperty(0, 1),
			},
			score0: int32Pointer(1), score1: int32Pointer(1),
			score0Deriv: true, score1Deriv: true,
		},
		{
			name:       "no score evidence stays unknown",
			properties: map[string]*rocket.Property{},
		},
		{
			name: "invalid goal team prevents score inference",
			properties: map[string]*rocket.Property{
				"Goals": matchGoalsProperty(2),
			},
		},
		{
			name: "explicit winner is not marked derived",
			properties: map[string]*rocket.Property{
				"Team0Score":  {Type: "IntProperty", Value: int32(2)},
				"Team1Score":  {Type: "IntProperty", Value: int32(1)},
				"WinningTeam": {Type: "IntProperty", Value: int32(1)},
			},
			score0: int32Pointer(2), score1: int32Pointer(1), winner: int32Pointer(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := (&rocket.Replay{Properties: tt.properties}).MatchData()
			if err != nil {
				t.Fatal(err)
			}
			assertOptionalInt32(t, "Team0Score", match.Team0Score, tt.score0)
			assertOptionalInt32(t, "Team1Score", match.Team1Score, tt.score1)
			assertOptionalInt32(t, "WinningTeam", match.WinningTeam, tt.winner)
			if match.Team0ScoreDerived != tt.score0Deriv || match.Team1ScoreDerived != tt.score1Deriv ||
				match.WinningTeamDerived != tt.winDeriv {
				t.Errorf("derived fields scores=%v/%v winner=%v; want %v/%v/%v",
					match.Team0ScoreDerived, match.Team1ScoreDerived, match.WinningTeamDerived,
					tt.score0Deriv, tt.score1Deriv, tt.winDeriv)
			}
			if match.Teams[0].IsWinner != (tt.winner != nil && *tt.winner == 0) ||
				match.Teams[1].IsWinner != (tt.winner != nil && *tt.winner == 1) {
				t.Errorf("unexpected winners: %+v", match.Teams)
			}
		})
	}
}

func TestReplayMatchDataMissingAndMalformedProperties(t *testing.T) {
	if _, err := (*rocket.Replay)(nil).MatchData(); err == nil || !strings.Contains(err.Error(), "replay is nil") {
		t.Fatalf("got %v, want nil replay error", err)
	}
	match, err := (&rocket.Replay{}).MatchData()
	if err != nil || len(match.Teams) != 2 || len(match.Goals) != 0 || len(match.PlayerStats) != 0 {
		t.Fatalf("empty replay match data: %+v, %v", match, err)
	}

	for _, tt := range []struct {
		name       string
		properties map[string]*rocket.Property
		want       string
	}{
		{
			name: "wrong map name type",
			properties: map[string]*rocket.Property{
				"MapName": {Type: "IntProperty", Value: int32(7)},
			},
			want: `property "MapName"`,
		},
		{
			name: "wrong player array type",
			properties: map[string]*rocket.Property{
				"PlayerStats": {Type: "IntProperty", Value: int32(7)},
			},
			want: `property "PlayerStats"`,
		},
		{
			name: "wrong goal field type",
			properties: map[string]*rocket.Property{
				"Goals": {
					Type: "ArrayProperty",
					Groups: []*rocket.PropertyGroup{{Properties: map[string]*rocket.Property{
						"frame": {Type: "StrProperty", Value: "bad"},
					}}},
				},
			},
			want: `property "frame"`,
		},
		{
			name: "missing goal team",
			properties: map[string]*rocket.Property{
				"Goals": {
					Type: "ArrayProperty",
					Groups: []*rocket.PropertyGroup{{Properties: map[string]*rocket.Property{
						"frame":      {Type: "IntProperty", Value: int32(7)},
						"PlayerName": {Type: "StrProperty", Value: "player"},
					}}},
				},
			},
			want: `required property "PlayerTeam" is missing`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&rocket.Replay{Properties: tt.properties}).MatchData()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestMatchDataSupportsReplayFixtures(t *testing.T) {
	for _, name := range []string{
		"first", "second", "third",
		"boxcars/no-frames", "boxcars/07e9", "boxcars/7083", "boxcars/d52eb",
		"boxcars/9a2cd", "boxcars/0ca5", "boxcars/3cdd", "boxcars/dropshot_rumble",
		"boxcars/heatseeker", "boxcars/MaxNumJumps",
	} {
		t.Run(name, func(t *testing.T) {
			replay, err := parsers.Parse(source.FromFile("samples/" + name + ".replay"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := replay.MatchData(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func int32Pointer(value int32) *int32 {
	return &value
}

func matchGoalsProperty(teams ...int32) *rocket.Property {
	groups := make([]*rocket.PropertyGroup, 0, len(teams))
	for i, team := range teams {
		groups = append(groups, &rocket.PropertyGroup{Properties: map[string]*rocket.Property{
			"frame":      {Type: "IntProperty", Value: int32(i)},
			"PlayerName": {Type: "StrProperty", Value: "player"},
			"PlayerTeam": {Type: "IntProperty", Value: team},
		}})
	}
	return &rocket.Property{Type: "ArrayProperty", Groups: groups}
}

func assertOptionalInt32(t *testing.T, name string, got, want *int32) {
	t.Helper()
	if got == nil || want == nil {
		if got != nil || want != nil {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
		return
	}
	if *got != *want {
		t.Errorf("%s: got %d, want %d", name, *got, *want)
	}
}
