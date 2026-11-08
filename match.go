package rocket

import "fmt"

// MatchData contains commonly used replay metadata in typed form.
type MatchData struct {
	MapName            string
	Date               string
	MatchType          string
	TeamSize           int32
	Team0Score         *int32
	Team1Score         *int32
	Team0ScoreDerived  bool // True when inferred from the Goals property.
	Team1ScoreDerived  bool // True when inferred from the Goals property.
	WinningTeam        *int32
	WinningTeamDerived bool // True when inferred from the normalized team scores.
	Goals              []Goal
	Highlights         []Highlight
	PlayerStats        []PlayerStat
	Teams              []Team
}

// Goal records the scorer, team, and frame for a goal.
type Goal struct {
	Frame      int32
	PlayerName string
	PlayerTeam int32
}

// Highlight identifies the actors involved in a replay highlight.
type Highlight struct {
	Frame         int32
	CarName       string
	BallName      string
	GoalActorName string
}

// PlayerStat contains a player's match results and identity metadata.
type PlayerStat struct {
	Name     string
	Team     *int32
	Score    int32
	Goals    int32
	Assists  int32
	Saves    int32
	Shots    int32
	IsBot    bool
	Platform string
	OnlineID *int64
}

// MatchData reads the common typed match metadata from the replay properties.
// The property maps remain available for fields not represented by MatchData.
func (replay *Replay) MatchData() (MatchData, error) {
	if replay == nil {
		return MatchData{}, fmt.Errorf("replay is nil")
	}

	properties := replay.Properties
	result := MatchData{}
	var err error
	if result.MapName, _, err = matchString(properties, "MapName"); err != nil {
		return MatchData{}, err
	}
	if result.Date, _, err = matchString(properties, "Date"); err != nil {
		return MatchData{}, err
	}
	if result.MatchType, _, err = matchString(properties, "MatchType"); err != nil {
		return MatchData{}, err
	}
	if result.TeamSize, _, err = matchInt32(properties, "TeamSize"); err != nil {
		return MatchData{}, err
	}
	if value, ok, err := matchInt32(properties, "Team0Score"); err != nil {
		return MatchData{}, err
	} else if ok {
		result.Team0Score = &value
	}
	if value, ok, err := matchInt32(properties, "Team1Score"); err != nil {
		return MatchData{}, err
	} else if ok {
		result.Team1Score = &value
	}
	if value, ok, err := matchInt32(properties, "WinningTeam"); err != nil {
		return MatchData{}, err
	} else if ok {
		result.WinningTeam = &value
	}

	if result.Goals, err = matchGoals(properties); err != nil {
		return MatchData{}, err
	}
	if _, hasGoals := properties["Goals"]; hasGoals {
		if team0Goals, team1Goals, complete := countGoalsByTeam(result.Goals); complete {
			if result.Team0Score == nil {
				result.Team0Score = &team0Goals
				result.Team0ScoreDerived = true
			}
			if result.Team1Score == nil {
				result.Team1Score = &team1Goals
				result.Team1ScoreDerived = true
			}
		}
	}
	if result.WinningTeam == nil && result.Team0Score != nil && result.Team1Score != nil {
		if *result.Team0Score > *result.Team1Score {
			value := int32(0)
			result.WinningTeam = &value
			result.WinningTeamDerived = true
		} else if *result.Team1Score > *result.Team0Score {
			value := int32(1)
			result.WinningTeam = &value
			result.WinningTeamDerived = true
		}
	}
	if result.Highlights, err = matchHighlights(properties); err != nil {
		return MatchData{}, err
	}
	if result.PlayerStats, err = matchPlayerStats(properties); err != nil {
		return MatchData{}, err
	}
	result.Teams = matchTeams(result)
	return result, nil
}

func countGoalsByTeam(goals []Goal) (int32, int32, bool) {
	var scores [2]int32
	for _, goal := range goals {
		if goal.PlayerTeam < 0 || goal.PlayerTeam > 1 {
			return 0, 0, false
		}
		scores[goal.PlayerTeam]++
	}
	return scores[0], scores[1], true
}

func matchString(properties map[string]*Property, name string) (string, bool, error) {
	property, ok := properties[name]
	if !ok {
		return "", false, nil
	}
	if property == nil {
		return "", true, fmt.Errorf("property %q is nil", name)
	}
	value, ok := property.Value.(string)
	if !ok {
		return "", true, fmt.Errorf("property %q has type %T, want string", name, property.Value)
	}
	return value, true, nil
}

func matchInt32(properties map[string]*Property, name string) (int32, bool, error) {
	property, ok := properties[name]
	if !ok {
		return 0, false, nil
	}
	if property == nil {
		return 0, true, fmt.Errorf("property %q is nil", name)
	}
	value, ok := property.Value.(int32)
	if !ok {
		return 0, true, fmt.Errorf("property %q has type %T, want int32", name, property.Value)
	}
	return value, true, nil
}

func matchBool(properties map[string]*Property, name string) (bool, bool, error) {
	property, ok := properties[name]
	if !ok {
		return false, false, nil
	}
	if property == nil {
		return false, true, fmt.Errorf("property %q is nil", name)
	}
	value, ok := property.Value.(bool)
	if !ok {
		return false, true, fmt.Errorf("property %q has type %T, want bool", name, property.Value)
	}
	return value, true, nil
}

func matchArray(properties map[string]*Property, name string) ([]*PropertyGroup, error) {
	property, ok := properties[name]
	if !ok {
		return nil, nil
	}
	if property == nil {
		return nil, fmt.Errorf("property %q is nil", name)
	}
	if property.Type != "ArrayProperty" {
		return nil, fmt.Errorf("property %q has type %q, want ArrayProperty", name, property.Type)
	}
	return property.Groups, nil
}

func requiredMatchString(properties map[string]*Property, name string) (string, error) {
	value, ok, err := matchString(properties, name)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("required property %q is missing", name)
	}
	return value, nil
}

func requiredMatchInt32(properties map[string]*Property, name string) (int32, error) {
	value, ok, err := matchInt32(properties, name)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("required property %q is missing", name)
	}
	return value, nil
}

func matchGoals(properties map[string]*Property) ([]Goal, error) {
	groups, err := matchArray(properties, "Goals")
	if err != nil {
		return nil, err
	}
	result := make([]Goal, 0, len(groups))
	for i, group := range groups {
		if group == nil {
			return nil, fmt.Errorf("Goals group %d is nil", i)
		}
		frame, err := requiredMatchInt32(group.Properties, "frame")
		if err != nil {
			return nil, fmt.Errorf("Goals group %d: %w", i, err)
		}
		playerName, err := requiredMatchString(group.Properties, "PlayerName")
		if err != nil {
			return nil, fmt.Errorf("Goals group %d: %w", i, err)
		}
		playerTeam, err := requiredMatchInt32(group.Properties, "PlayerTeam")
		if err != nil {
			return nil, fmt.Errorf("Goals group %d: %w", i, err)
		}
		result = append(result, Goal{Frame: frame, PlayerName: playerName, PlayerTeam: playerTeam})
	}
	return result, nil
}

func matchHighlights(properties map[string]*Property) ([]Highlight, error) {
	groups, err := matchArray(properties, "HighLights")
	if err != nil {
		return nil, err
	}
	result := make([]Highlight, 0, len(groups))
	for i, group := range groups {
		if group == nil {
			return nil, fmt.Errorf("HighLights group %d is nil", i)
		}
		frame, err := requiredMatchInt32(group.Properties, "frame")
		if err != nil {
			return nil, fmt.Errorf("HighLights group %d: %w", i, err)
		}
		carName, err := requiredMatchString(group.Properties, "CarName")
		if err != nil {
			return nil, fmt.Errorf("HighLights group %d: %w", i, err)
		}
		ballName, err := requiredMatchString(group.Properties, "BallName")
		if err != nil {
			return nil, fmt.Errorf("HighLights group %d: %w", i, err)
		}
		goalName, _, err := matchString(group.Properties, "GoalActorName")
		if err != nil {
			return nil, fmt.Errorf("HighLights group %d: %w", i, err)
		}
		result = append(result, Highlight{
			Frame: frame, CarName: carName, BallName: ballName, GoalActorName: goalName,
		})
	}
	return result, nil
}

func matchPlayerStats(properties map[string]*Property) ([]PlayerStat, error) {
	groups, err := matchArray(properties, "PlayerStats")
	if err != nil {
		return nil, err
	}
	result := make([]PlayerStat, 0, len(groups))
	for i, group := range groups {
		if group == nil {
			return nil, fmt.Errorf("PlayerStats group %d is nil", i)
		}
		fields := group.Properties
		stat := PlayerStat{}
		if stat.Name, _, err = matchString(fields, "Name"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		}
		if stat.Platform, _, err = matchString(fields, "Platform"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		}
		if value, ok, err := matchInt32(fields, "Team"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		} else if ok {
			stat.Team = &value
		}
		if value, _, err := matchInt32(fields, "Score"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		} else {
			stat.Score = value
		}
		if stat.Goals, _, err = matchInt32(fields, "Goals"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		}
		if stat.Assists, _, err = matchInt32(fields, "Assists"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		}
		if stat.Saves, _, err = matchInt32(fields, "Saves"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		}
		if stat.Shots, _, err = matchInt32(fields, "Shots"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		}
		if stat.IsBot, _, err = matchBool(fields, "bBot"); err != nil {
			return nil, fmt.Errorf("PlayerStats group %d: %w", i, err)
		}
		if property, ok := fields["OnlineID"]; ok {
			if property == nil {
				return nil, fmt.Errorf("PlayerStats group %d: property %q is nil", i, "OnlineID")
			}
			value, ok := property.Value.(int64)
			if !ok {
				return nil, fmt.Errorf("PlayerStats group %d: property %q has type %T, want int64", i, "OnlineID", property.Value)
			}
			stat.OnlineID = &value
		}
		result = append(result, stat)
	}
	return result, nil
}

func matchTeams(data MatchData) []Team {
	teams := []Team{
		{Name: "Blue", Index: 0},
		{Name: "Orange", Index: 1},
	}
	if data.Team0Score != nil {
		teams[0].Score = int(*data.Team0Score)
		teams[0].HasScore = true
	}
	if data.Team1Score != nil {
		teams[1].Score = int(*data.Team1Score)
		teams[1].HasScore = true
	}
	for _, player := range data.PlayerStats {
		if player.Team == nil || *player.Team < 0 || *player.Team > 1 {
			continue
		}
		team := &teams[*player.Team]
		team.Players = append(team.Players, player.Name)
		team.PlayerStats = append(team.PlayerStats, player)
	}
	teams[0].ScoreDerived = data.Team0ScoreDerived
	teams[1].ScoreDerived = data.Team1ScoreDerived
	if data.WinningTeam != nil {
		if *data.WinningTeam == 0 || *data.WinningTeam == 1 {
			teams[*data.WinningTeam].IsWinner = true
			teams[*data.WinningTeam].WinnerDerived = data.WinningTeamDerived
		}
	} else if data.Team0Score != nil && data.Team1Score != nil {
		if *data.Team0Score > *data.Team1Score {
			teams[0].IsWinner = true
		} else if *data.Team1Score > *data.Team0Score {
			teams[1].IsWinner = true
		}
	}
	return teams
}
