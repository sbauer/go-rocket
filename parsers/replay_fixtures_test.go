package parsers

import (
	"reflect"
	"testing"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

type fixtureUpdate struct {
	frame, actor, stream int
	name                 string
	value                interface{}
}

func TestAdditionalReplayFixtures(t *testing.T) {
	tests := []struct {
		name                string
		licensee, net       int
		mapName, build      string
		frames              int
		spawns, deletes     int
		updates             int
		ids                 []byte
		psyNetExtraBytes    int
		objects             []string
		referenceAttributes []fixtureUpdate
	}{
		{
			name: "07e9", licensee: 11, net: 0, mapName: "HoopsStadium_P", build: "160513.59355.132594",
			frames: 319, spawns: 30, deletes: 0, updates: 627, ids: []byte{1},
			objects: []string{"Archetypes.Ball.Ball_Basketball", "Archetypes.GameEvent.GameEvent_BasketballSplitscreen"},
			referenceAttributes: []fixtureUpdate{
				{0, 1, 43, "TAGame.GameEvent_Soccar_TA:SecondsRemaining", int32(300)},
			},
		},
		{
			name: "7083", licensee: 20, net: 2, mapName: "HoopsStadium_P", build: "171122.50648.178784",
			frames: 8346, spawns: 2786, deletes: 460, updates: 46988, ids: []byte{0, 1, 6},
			objects: []string{"Archetypes.Ball.Ball_BasketBall", "Archetypes.GameEvent.GameEvent_Basketball"},
		},
		{
			name: "d52eb", licensee: 23, net: 8, mapName: "cs_day_p", build: "181119.52056.216417",
			frames: 17815, spawns: 2852, deletes: 415, updates: 58247, ids: []byte{1, 4, 7}, psyNetExtraBytes: 24,
			objects: []string{"Archetypes.Ball.Ball_Default"},
		},
		{
			name: "9a2cd", licensee: 26, net: 10, mapName: "Haunted_TrainStation_P", build: "190605.83642.236736",
			frames: 2616, spawns: 1097, deletes: 273, updates: 19309, ids: []byte{0, 1, 2, 4, 7},
			objects: []string{"Archetypes.Ball.Ball_Haunted", "Archetypes.SpecialPickups.SpecialPickup_HauntedBallBeam"},
		},
		{
			name: "0ca5", licensee: 27, net: 10, mapName: "chn_stadium_P", build: "191011.52816.250425",
			frames: 472, spawns: 37, deletes: 1, updates: 991, ids: []byte{5},
			objects: []string{"Archetypes.Ball.Ball_Trajectory"},
		},
		{
			name: "3cdd", licensee: 32, net: 10, mapName: "Stadium_Winter_P", build: "250505.52219.483820",
			frames: 753, spawns: 265, deletes: 28, updates: 1946, ids: []byte{1},
			objects: []string{
				"Archetypes.Ball.Ball_Puck",
				"GameInfo_SnowDayTerritory.GameInfo.GameInfo_SnowDayTerritory:Archetype",
				"Archetypes.SpecialPickups.BM.SpecialPickup_BallFreeze_BM",
			},
			referenceAttributes: []fixtureUpdate{
				{208, 51, 23, "TAGame.CarComponent_AirActivate_TA:AirActivateCount", int32(1)},
				{492, 68, 23, "TAGame.CarComponent_AirActivate_TA:AirActivateCount", int32(1)},
			},
		},
		{
			name: "dropshot_rumble", licensee: 32, net: 10, mapName: "UF_Day_P", build: "250307.56620.477186",
			frames: 468, spawns: 36, deletes: 0, updates: 679, ids: []byte{1},
			objects: []string{"Archetypes.Ball.Ball_Breakout", "GameInfo_LTM_DropshotRumble.GameInfo.GameInfo_LTM_DropshotRumble:Archetype"},
			referenceAttributes: []fixtureUpdate{
				{275, 1, 68, "TAGame.Ball_Breakout_TA:LastTeamTouch", byte(1)},
			},
		},
		{
			name: "heatseeker", licensee: 32, net: 10, mapName: "UF_Day_P", build: "250307.56620.477186",
			frames: 726, spawns: 51, deletes: 0, updates: 736, ids: []byte{1},
			objects: []string{"Archetypes.Ball.Ball_God", "GameInfo_HeatseekerTerritory.GameInfo.GameInfo_HeatseekerTerritory:Archetype"},
			referenceAttributes: []fixtureUpdate{
				{612, 0, 66, "TAGame.Ball_God_TA:TargetSpeed", float32(2985)},
			},
		},
		{
			name: "MaxNumJumps", licensee: 34, net: 12, mapName: "UtopiaStadium_P", build: "261001.2867.529371",
			frames: 6065, spawns: 1804, deletes: 327, updates: 33672, ids: []byte{11},
			objects: []string{"Archetypes.Ball.Ball_Default"},
			referenceAttributes: []fixtureUpdate{
				{0, 42, 70, "TAGame.Car_TA:MaxNumJumps", int32(3)},
				{94, 42, 75, "TAGame.Car_TA:bUnlimitedJumps", true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replay, err := ParseWithOptions(source.FromFile("../samples/boxcars/"+tt.name+".replay"),
				ParseOptions{DecodeNetwork: true})
			if err != nil {
				t.Fatal(err)
			}
			header := replay.HeaderInfo
			if header.EngineVersion != 868 || header.LicenseeVersion != tt.licensee ||
				header.NetVersion != tt.net || header.ClassName != "TAGame.Replay_Soccar_TA" {
				t.Fatalf("unexpected header: %+v", header)
			}
			for name, want := range map[string]interface{}{
				"MapName": tt.mapName, "BuildVersion": tt.build, "NumFrames": int32(tt.frames),
			} {
				if property := replay.Properties[name]; property == nil || property.Value != want {
					t.Errorf("%s: got %+v, want %v", name, property, want)
				}
			}
			if len(replay.Frames) != tt.frames {
				t.Fatalf("got %d decoded frames, want %d", len(replay.Frames), tt.frames)
			}

			seenIDs := make(map[byte]bool)
			seenObjects := make(map[string]bool)
			var spawns, deletes, updates int
			checkID := func(id rocket.UniqueID) {
				t.Helper()
				seenIDs[id.SystemID] = true
				switch id.SystemID {
				case 0:
					if _, ok := id.RemoteID.(uint32); !ok {
						t.Errorf("split-screen ID has type %T", id.RemoteID)
					}
				case 1, 4, 5:
					if _, ok := id.RemoteID.(uint64); !ok {
						t.Errorf("system %d ID has type %T", id.SystemID, id.RemoteID)
					}
				case 2:
					if value, ok := id.RemoteID.(rocket.PlayStationID); !ok || len(value.Unknown) != 16 {
						t.Errorf("unexpected PlayStation ID: %#v", id.RemoteID)
					}
				case 6:
					if value, ok := id.RemoteID.(rocket.SwitchID); !ok || len(value.Unknown) != 24 {
						t.Errorf("unexpected Switch ID: %#v", id.RemoteID)
					}
				case 7:
					if value, ok := id.RemoteID.(rocket.PsyNetID); !ok || len(value.Unknown) != tt.psyNetExtraBytes {
						t.Errorf("unexpected PsyNet ID: %#v", id.RemoteID)
					}
				case 11:
					if _, ok := id.RemoteID.(string); !ok {
						t.Errorf("Epic ID has type %T", id.RemoteID)
					}
				default:
					t.Errorf("unexpected platform %d", id.SystemID)
				}
			}
			for _, frame := range replay.Frames {
				for _, actor := range frame.Actors {
					if actor.Spawn != nil {
						spawns++
						seenObjects[replay.Objects[actor.Spawn.ObjectIndex]] = true
					}
					if actor.Deleted {
						deletes++
					}
					for _, attribute := range actor.Attributes {
						updates++
						switch value := attribute.Value.(type) {
						case rocket.UniqueID:
							checkID(value)
						case *rocket.UniqueID:
							if value != nil {
								checkID(*value)
							}
						case rocket.Reservation:
							checkID(value.UniqueID)
						}
					}
				}
			}
			if spawns != tt.spawns || deletes != tt.deletes || updates != tt.updates {
				t.Errorf("got %d spawns, %d deletions, %d updates; want %d, %d, %d",
					spawns, deletes, updates, tt.spawns, tt.deletes, tt.updates)
			}
			expectedIDs := make(map[byte]bool)
			for _, id := range tt.ids {
				expectedIDs[id] = true
			}
			if !reflect.DeepEqual(seenIDs, expectedIDs) {
				t.Errorf("platform IDs: got %v, want %v", seenIDs, expectedIDs)
			}
			for _, object := range tt.objects {
				if !seenObjects[object] {
					t.Errorf("missing spawned object %q", object)
				}
			}
			for _, expected := range tt.referenceAttributes {
				assertFixtureUpdate(t, replay, expected)
			}
			if tt.name == "MaxNumJumps" {
				assertNestedPlayerID(t, replay)
			}
		})
	}
}

func assertFixtureUpdate(t *testing.T, replay *rocket.Replay, want fixtureUpdate) {
	t.Helper()
	for _, actor := range replay.Frames[want.frame].Actors {
		if actor.ActorID != int32(want.actor) {
			continue
		}
		for _, attribute := range actor.Attributes {
			if attribute.StreamID == int32(want.stream) && replay.Objects[attribute.ObjectIndex] == want.name {
				if !reflect.DeepEqual(attribute.Value, want.value) {
					t.Errorf("frame %d actor %d %s: got %#v, want %#v",
						want.frame, want.actor, want.name, attribute.Value, want.value)
				}
				return
			}
		}
	}
	t.Errorf("frame %d actor %d missing stream %d %s", want.frame, want.actor, want.stream, want.name)
}

func assertNestedPlayerID(t *testing.T, replay *rocket.Replay) {
	t.Helper()
	stats := replay.Properties["PlayerStats"]
	if stats == nil || len(stats.Groups) != 2 {
		t.Fatalf("unexpected player stats: %+v", stats)
	}
	property := stats.Groups[0].Properties["PlayerID"]
	for _, step := range []struct {
		name, field string
	}{
		{"UniqueNetId", "NpId"},
		{"SceNpId", "Handle"},
		{"SceNpOnlineId", "Data"},
	} {
		if property == nil {
			t.Fatalf("missing nested %s property", step.name)
		}
		value, ok := property.Value.(rocket.StructProperty)
		if !ok || value.Name != step.name {
			t.Fatalf("got %#v, want struct %s", property.Value, step.name)
		}
		property = value.Fields[step.field]
	}
	if property == nil || property.Value != uint64(0) {
		t.Fatalf("unexpected nested Data property: %+v", property)
	}
}
