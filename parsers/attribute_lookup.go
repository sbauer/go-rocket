package parsers

import (
	"fmt"
	"strings"

	"github.com/sbauer/go-rocket"
)

type classAttributes struct {
	classObjectIndex int32
	streams          map[int32]int32
	maxStreamID      int32 // -1 when the class has no attributes.
}

type attributeLookup struct {
	objects       []string
	classIndices  map[string]int32
	classes       map[int32]*classAttributes
	classesByLeaf map[string][]string
}

// Cache IDs are not unique in replay files, so known class relationships take
// precedence over the serialized parent IDs.
var knownClassParents = map[string]string{
	"Core.Object":                              "",
	"Engine.Actor":                             "Core.Object",
	"Engine.Info":                              "Engine.Actor",
	"Engine.ReplicationInfo":                   "Engine.Info",
	"Engine.GameReplicationInfo":               "Engine.ReplicationInfo",
	"Engine.PlayerReplicationInfo":             "Engine.ReplicationInfo",
	"Engine.TeamInfo":                          "Engine.ReplicationInfo",
	"Engine.Pawn":                              "Engine.Actor",
	"ProjectX.GRI_X":                           "Engine.GameReplicationInfo",
	"ProjectX.NetModeReplicator_X":             "Engine.ReplicationInfo",
	"ProjectX.PRI_X":                           "Engine.PlayerReplicationInfo",
	"ProjectX.Pawn_X":                          "Engine.Pawn",
	"TAGame.GRI_TA":                            "ProjectX.GRI_X",
	"TAGame.PRI_TA":                            "ProjectX.PRI_X",
	"TAGame.PRI_Breakout_TA":                   "TAGame.PRI_TA",
	"TAGame.CameraSettingsActor_TA":            "Engine.Actor",
	"TAGame.Cannon_TA":                         "Engine.Actor",
	"TAGame.GameEvent_TA":                      "Engine.ReplicationInfo",
	"TAGame.GameEvent_Team_TA":                 "TAGame.GameEvent_TA",
	"TAGame.GameEvent_Soccar_TA":               "TAGame.GameEvent_Team_TA",
	"TAGame.GameEvent_SoccarPrivate_TA":        "TAGame.GameEvent_Soccar_TA",
	"TAGame.GameEvent_SoccarSplitscreen_TA":    "TAGame.GameEvent_SoccarPrivate_TA",
	"TAGame.GameEvent_Breakout_TA":             "TAGame.GameEvent_Soccar_TA",
	"TAGame.GameEvent_GodBall_TA":              "TAGame.GameEvent_Soccar_TA",
	"TAGame.GameEvent_Territory_TA":            "TAGame.GameEvent_Soccar_TA",
	"TAGame.GameEvent_Tutorial_TA":             "TAGame.GameEvent_Soccar_TA",
	"TAGame.GameEvent_Training_TA":             "TAGame.GameEvent_Soccar_TA",
	"TAGame.GameEvent_Training_Goalie_TA":      "TAGame.GameEvent_Training_TA",
	"TAGame.Team_TA":                           "Engine.TeamInfo",
	"TAGame.Team_Soccar_TA":                    "TAGame.Team_TA",
	"TAGame.RBActor_TA":                        "ProjectX.Pawn_X",
	"TAGame.Ball_TA":                           "TAGame.RBActor_TA",
	"TAGame.Ball_Breakout_TA":                  "TAGame.Ball_TA",
	"TAGame.Ball_Fire_TA":                      "TAGame.Ball_TA",
	"TAGame.Ball_God_TA":                       "TAGame.Ball_TA",
	"TAGame.Ball_Haunted_TA":                   "TAGame.Ball_TA",
	"TAGame.Ball_Trajectory_TA":                "TAGame.Ball_TA",
	"TAGame.Ball_Tutorial_TA":                  "TAGame.Ball_TA",
	"TAGame.Vehicle_TA":                        "TAGame.RBActor_TA",
	"TAGame.Car_TA":                            "TAGame.Vehicle_TA",
	"TAGame.CarComponent_TA":                   "Engine.Actor",
	"TAGame.CarComponent_AirActivate_TA":       "TAGame.CarComponent_TA",
	"TAGame.CarComponent_Boost_TA":             "TAGame.CarComponent_AirActivate_TA",
	"TAGame.CarComponent_Jump_TA":              "TAGame.CarComponent_TA",
	"TAGame.CarComponent_DoubleJump_TA":        "TAGame.CarComponent_AirActivate_TA",
	"TAGame.CarComponent_Dodge_TA":             "TAGame.CarComponent_AirActivate_TA",
	"TAGame.CarComponent_FlipCar_TA":           "TAGame.CarComponent_TA",
	"TAGame.CarComponent_TerritoryDemolish_TA": "TAGame.CarComponent_TA",
	"TAGame.PickupTimer_TA":                    "TAGame.CarComponent_TA",
	"TAGame.SpecialPickup_TA":                  "TAGame.CarComponent_TA",
	"TAGame.SpecialPickup_BallGravity_TA":      "TAGame.SpecialPickup_TA",
	"TAGame.SpecialPickup_Targeted_TA":         "TAGame.SpecialPickup_TA",
	"TAGame.SpecialPickup_BallFreeze_TA":       "TAGame.SpecialPickup_Targeted_TA",
	"TAGame.SpecialPickup_HauntedBallBeam_TA":  "TAGame.SpecialPickup_BallGravity_TA",
	"TAGame.CrowdActor_TA":                     "Engine.Actor",
	"TAGame.CrowdManager_TA":                   "Engine.Actor",
	"TAGame.DynamicMeshActor_TA":               "Engine.Actor",
	"TAGame.HauntedBallTrapTrigger_TA":         "TAGame.DynamicMeshActor_TA",
	"TAGame.TrackerWallDynamicMeshActor_TA":    "TAGame.DynamicMeshActor_TA",
	"TAGame.RumblePickups_TA":                  "Engine.Actor",
	"TAGame.ViralItemActor_TA":                 "Engine.Actor",
	"TAGame.VehiclePickup_TA":                  "Engine.Actor",
	"TAGame.VehiclePickup_Boost_TA":            "TAGame.VehiclePickup_TA",
}

var knownArchetypeClasses = map[string]string{
	"Archetypes.Ball.Ball_Default":                                                                    "TAGame.Ball_TA",
	"Archetypes.Ball.Ball_Basketball":                                                                 "TAGame.Ball_TA",
	"Archetypes.Ball.Ball_BasketBall":                                                                 "TAGame.Ball_TA",
	"Archetypes.Ball.Ball_Beachball":                                                                  "TAGame.Ball_TA",
	"Archetypes.Ball.Ball_Fire":                                                                       "TAGame.Ball_Fire_TA",
	"Archetypes.Ball.ball_luminousairplane":                                                           "TAGame.Ball_TA",
	"Archetypes.Ball.Ball_Puck":                                                                       "TAGame.Ball_TA",
	"Archetypes.Ball.Ball_Training":                                                                   "TAGame.Ball_Tutorial_TA",
	"Archetypes.Ball.Ball_Breakout":                                                                   "TAGame.Ball_Breakout_TA",
	"Archetypes.Ball.Ball_God":                                                                        "TAGame.Ball_God_TA",
	"Archetypes.Ball.Ball_Haunted":                                                                    "TAGame.Ball_Haunted_TA",
	"Archetypes.Ball.Ball_Trajectory":                                                                 "TAGame.Ball_Trajectory_TA",
	"Archetypes.Car.Car_Default":                                                                      "TAGame.Car_TA",
	"Archetypes.CarComponents.CarComponent_Boost":                                                     "TAGame.CarComponent_Boost_TA",
	"Archetypes.CarComponents.CarComponent_Dodge":                                                     "TAGame.CarComponent_Dodge_TA",
	"Archetypes.CarComponents.CarComponent_DoubleJump":                                                "TAGame.CarComponent_DoubleJump_TA",
	"Archetypes.CarComponents.CarComponent_FlipCar":                                                   "TAGame.CarComponent_FlipCar_TA",
	"Archetypes.CarComponents.CarComponent_Jump":                                                      "TAGame.CarComponent_Jump_TA",
	"Archetypes.CarComponents.CarComponent_TerritoryDemolish":                                         "TAGame.CarComponent_TerritoryDemolish_TA",
	"Archetypes.GameEvent.GameEvent_Soccar":                                                           "TAGame.GameEvent_Soccar_TA",
	"Archetypes.GameEvent.GameEvent_Basketball":                                                       "TAGame.GameEvent_Soccar_TA",
	"Archetypes.GameEvent.GameEvent_BasketballSplitscreen":                                            "TAGame.GameEvent_SoccarSplitscreen_TA",
	"Archetypes.Tutorial.Cannon":                                                                      "TAGame.Cannon_TA",
	"Archetypes.SpecialPickups.BM.SpecialPickup_BallFreeze_BM":                                        "TAGame.SpecialPickup_BallFreeze_TA",
	"Archetypes.SpecialPickups.SpecialPickup_HauntedBallBeam":                                         "TAGame.SpecialPickup_HauntedBallBeam_TA",
	"Archetypes.Teams.Team0":                                                                          "TAGame.Team_Soccar_TA",
	"Archetypes.Teams.Team1":                                                                          "TAGame.Team_Soccar_TA",
	"GameInfo_Soccar.GameInfo.GameInfo_Soccar:GameReplicationInfoArchetype":                           "TAGame.GRI_TA",
	"GameInfo_Basketball.GameInfo.GameInfo_Basketball:GameReplicationInfoArchetype":                   "TAGame.GRI_TA",
	"GameInfo_Basketball.GameInfo.GameInfo_Basketball:Archetype":                                      "TAGame.GameEvent_Soccar_TA",
	"GameInfo_LTM_BeachBall.GameInfo.GameInfo_LTM_BeachBall:Archetype":                                "TAGame.GameEvent_Soccar_TA",
	"GameInfo_LTM_BeachBall.GameInfo.GameInfo_LTM_BeachBall:GameReplicationInfoArchetype":             "TAGame.GRI_TA",
	"GameInfo_Tutorial.GameInfo.GameInfo_Tutorial:GameReplicationInfoArchetype":                       "TAGame.GRI_TA",
	"GameInfo_Tutorial.GameEvent.GameEvent_Tutorial_Goalie":                                           "TAGame.GameEvent_Training_Goalie_TA",
	"GameInfo_HeatseekerTerritory.GameInfo.GameInfo_HeatseekerTerritory:Archetype":                    "TAGame.GameEvent_Soccar_TA",
	"GameInfo_HeatseekerTerritory.GameInfo.GameInfo_HeatseekerTerritory:GameReplicationInfoArchetype": "TAGame.GRI_TA",
	"GameInfo_LTM_DropshotRumble.GameInfo.GameInfo_LTM_DropshotRumble:Archetype":                      "TAGame.GameEvent_Soccar_TA",
	"GameInfo_LTM_DropshotRumble.GameInfo.GameInfo_LTM_DropshotRumble:GameReplicationInfoArchetype":   "TAGame.GRI_TA",
	"GameInfo_SnowDayTerritory.GameInfo.GameInfo_SnowDayTerritory:Archetype":                          "TAGame.GameEvent_Territory_TA",
	"GameInfo_SnowDayTerritory.GameInfo.GameInfo_SnowDayTerritory:GameReplicationInfoArchetype":       "TAGame.GRI_TA",
	"GameInfo_Territory.GameInfo.GameInfo_Territory:Archetype":                                        "TAGame.GameEvent_Territory_TA",
	"GameInfo_Territory.GameInfo.GameInfo_Territory:GameReplicationInfoArchetype":                     "TAGame.GRI_TA",
}

type attributeLookupBuilder struct {
	objects  []string
	byName   map[string]*rocket.ClassNetCache
	byID     map[int32][]*rocket.ClassNetCache
	visiting map[*rocket.ClassNetCache]bool
	resolved map[*rocket.ClassNetCache]*classAttributes
}

func newAttributeLookup(replay *rocket.Replay) (*attributeLookup, error) {
	if replay == nil {
		return nil, fmt.Errorf("replay is nil")
	}
	lookup := &attributeLookup{
		objects:       replay.Objects,
		classIndices:  make(map[string]int32),
		classes:       make(map[int32]*classAttributes),
		classesByLeaf: make(map[string][]string),
	}
	addClass := func(name string, index int32) error {
		if index < 0 || int(index) >= len(replay.Objects) {
			return fmt.Errorf("class %q object index %d out of range", name, index)
		}
		if replay.Objects[index] != name {
			return fmt.Errorf("class %q object index %d names %q", name, index, replay.Objects[index])
		}
		if previous, ok := lookup.classIndices[name]; ok {
			if previous != index {
				return fmt.Errorf("class %q has conflicting object indexes %d and %d", name, previous, index)
			}
			return nil
		}
		lookup.classIndices[name] = index
		leaf := name
		if separator := strings.LastIndexByte(name, '.'); separator >= 0 {
			leaf = name[separator+1:]
		}
		lookup.classesByLeaf[leaf] = append(lookup.classesByLeaf[leaf], name)
		return nil
	}
	for i, class := range replay.ClassIndices {
		if class == nil {
			return nil, fmt.Errorf("class index %d is nil", i)
		}
		if err := addClass(class.Class, class.Index); err != nil {
			return nil, err
		}
	}

	builder := &attributeLookupBuilder{
		objects:  replay.Objects,
		byName:   make(map[string]*rocket.ClassNetCache),
		byID:     make(map[int32][]*rocket.ClassNetCache),
		visiting: make(map[*rocket.ClassNetCache]bool),
		resolved: make(map[*rocket.ClassNetCache]*classAttributes),
	}
	for i, cache := range replay.NetCache {
		if cache == nil {
			return nil, fmt.Errorf("network cache %d is nil", i)
		}
		if cache.ObjectIndex < 0 || int(cache.ObjectIndex) >= len(replay.Objects) {
			return nil, fmt.Errorf("network cache %d object index %d out of range", i, cache.ObjectIndex)
		}
		name := replay.Objects[cache.ObjectIndex]
		if err := addClass(name, cache.ObjectIndex); err != nil {
			return nil, err
		}
		if _, exists := builder.byName[name]; exists {
			return nil, fmt.Errorf("duplicate network cache for class %q", name)
		}
		seenStreams := make(map[int32]bool)
		for j, property := range cache.Properties {
			if property == nil {
				return nil, fmt.Errorf("class %q property %d is nil", name, j)
			}
			if property.ObjectIndex < 0 || int(property.ObjectIndex) >= len(replay.Objects) {
				return nil, fmt.Errorf("class %q property %d object index %d out of range", name, j, property.ObjectIndex)
			}
			if property.StreamID < 0 {
				return nil, fmt.Errorf("class %q property %d has negative stream ID %d", name, j, property.StreamID)
			}
			if seenStreams[property.StreamID] {
				return nil, fmt.Errorf("class %q has duplicate stream ID %d", name, property.StreamID)
			}
			seenStreams[property.StreamID] = true
		}
		builder.byName[name] = cache
		builder.byID[cache.CacheID] = append(builder.byID[cache.CacheID], cache)
	}

	for _, cache := range replay.NetCache {
		attributes, err := builder.resolve(cache)
		if err != nil {
			return nil, err
		}
		lookup.classes[cache.ObjectIndex] = attributes
	}
	return lookup, nil
}

func (builder *attributeLookupBuilder) resolve(cache *rocket.ClassNetCache) (*classAttributes, error) {
	if attributes, ok := builder.resolved[cache]; ok {
		return attributes, nil
	}
	name := builder.objects[cache.ObjectIndex]
	if builder.visiting[cache] {
		return nil, fmt.Errorf("class %q has a parent cycle", name)
	}
	builder.visiting[cache] = true
	defer delete(builder.visiting, cache)

	parent, err := builder.parent(cache)
	if err != nil {
		return nil, fmt.Errorf("class %q: %w", name, err)
	}
	attributes := &classAttributes{
		classObjectIndex: cache.ObjectIndex,
		streams:          make(map[int32]int32),
		maxStreamID:      -1,
	}
	if parent != nil {
		inherited, err := builder.resolve(parent)
		if err != nil {
			return nil, fmt.Errorf("class %q: %w", name, err)
		}
		for stream, object := range inherited.streams {
			attributes.streams[stream] = object
		}
		attributes.maxStreamID = inherited.maxStreamID
	}
	for _, property := range cache.Properties {
		attributes.streams[property.StreamID] = property.ObjectIndex
		if property.StreamID > attributes.maxStreamID {
			attributes.maxStreamID = property.StreamID
		}
	}
	builder.resolved[cache] = attributes
	return attributes, nil
}

func (builder *attributeLookupBuilder) parent(cache *rocket.ClassNetCache) (*rocket.ClassNetCache, error) {
	name := builder.objects[cache.ObjectIndex]
	if parentName, known := knownClassParents[name]; known {
		for parentName != "" {
			if parent, ok := builder.byName[parentName]; ok {
				return parent, nil
			}
			if parentName != "TAGame.CarComponent_AirActivate_TA" {
				return nil, fmt.Errorf("missing parent class %q", parentName)
			}
			parentName = knownClassParents[parentName]
		}
		return nil, nil
	}
	if cache.CacheID == 0 && cache.ParentID == 0 && len(builder.byID[0]) == 1 {
		return nil, nil
	}
	var parent *rocket.ClassNetCache
	for _, candidate := range builder.byID[cache.ParentID] {
		if candidate == cache {
			continue
		}
		if parent != nil {
			return nil, fmt.Errorf("ambiguous parent cache ID %d", cache.ParentID)
		}
		parent = candidate
	}
	if parent == nil {
		return nil, fmt.Errorf("missing parent cache ID %d", cache.ParentID)
	}
	return parent, nil
}

func (lookup *attributeLookup) forObject(index int32) (*classAttributes, error) {
	if index < 0 || int(index) >= len(lookup.objects) {
		return nil, fmt.Errorf("object index %d out of range", index)
	}
	if attributes, ok := lookup.classes[index]; ok {
		return attributes, nil
	}
	name := lookup.objects[index]
	if classIndex, ok := lookup.classIndices[name]; ok && classIndex == index {
		return lookup.forClass(name)
	}
	if class, ok := knownArchetypeClasses[name]; ok {
		return lookup.forClass(class)
	}
	if separator := strings.IndexByte(name, '.'); separator >= 0 {
		leaf := name[separator+1:]
		if strings.HasPrefix(leaf, "Default__") {
			return lookup.forClass(name[:separator+1] + strings.TrimPrefix(leaf, "Default__"))
		}
	}
	if strings.Contains(name, ":PersistentLevel.") {
		leaf := name[strings.LastIndexByte(name, '.')+1:]
		separator := strings.LastIndexByte(leaf, '_')
		if separator >= 0 && isDecimal(leaf[separator+1:]) {
			classes := lookup.classesByLeaf[leaf[:separator]]
			if len(classes) == 1 {
				return lookup.forClass(classes[0])
			}
			if len(classes) > 1 {
				return nil, fmt.Errorf("object %d %q has ambiguous class %q", index, name, leaf[:separator])
			}
		}
	}
	return nil, fmt.Errorf("object %d %q has no known class", index, name)
}

func (lookup *attributeLookup) forClass(name string) (*classAttributes, error) {
	if index, ok := lookup.classIndices[name]; ok {
		if attributes, ok := lookup.classes[index]; ok {
			return attributes, nil
		}
	}
	if name == "TAGame.CarComponent_AirActivate_TA" {
		return lookup.forClass(knownClassParents[name])
	}
	return nil, fmt.Errorf("class %q has no network cache", name)
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
