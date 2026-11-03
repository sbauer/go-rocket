package parsers

import (
	"strings"
	"testing"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

func TestAttributeLookupReplayFixtures(t *testing.T) {
	for _, sample := range []string{"first", "second", "third"} {
		t.Run(sample, func(t *testing.T) {
			replay, err := Parse(source.FromFile("../samples/" + sample + ".replay"))
			if err != nil {
				t.Fatal(err)
			}
			lookup, err := newAttributeLookup(replay)
			if err != nil {
				t.Fatal(err)
			}

			root := fixtureAttributes(t, replay, lookup, "Core.Object")
			if len(root.streams) != 0 || root.maxStreamID != -1 {
				t.Fatalf("unexpected root attributes: %+v", root)
			}
			actor := fixtureAttributes(t, replay, lookup, "Engine.Actor")
			assertAttribute(t, replay, actor, 0, "Engine.Actor:RelativeRotation")
			assertAttribute(t, replay, actor, 19, "Engine.Actor:Location")
			if actor.maxStreamID != 19 {
				t.Fatalf("actor maximum stream ID: got %d", actor.maxStreamID)
			}

			car := fixtureAttributes(t, replay, lookup, "TAGame.Car_TA")
			assertAttribute(t, replay, car, 0, "Engine.Actor:RelativeRotation")
			assertAttribute(t, replay, car, 20, "Engine.Pawn:RootMotionInterpCurveLastValue")
			assertAttribute(t, replay, car, 38, "TAGame.RBActor_TA:WeldedInfo")
			assertAttribute(t, replay, car, 45, "TAGame.Vehicle_TA:ReplicatedSteer")
			assertAttribute(t, replay, car, 50, "TAGame.Car_TA:AddedCarForceMultiplier")
			assertAttribute(t, replay, car, 59, "TAGame.Car_TA:ServerEquipProduct")
			if car.maxStreamID != 59 {
				t.Fatalf("car maximum stream ID: got %d", car.maxStreamID)
			}

			pri := fixtureAttributes(t, replay, lookup, "TAGame.PRI_TA")
			assertAttribute(t, replay, pri, 20, "Engine.PlayerReplicationInfo:RemoteUserData")
			assertAttribute(t, replay, pri, 39, "Engine.PlayerReplicationInfo:ServerUpdateRemoteUserData")
			assertAttribute(t, replay, pri, 40, "TAGame.PRI_TA:StayAsPartyVoteYes")
			assertAttribute(t, replay, pri, 107, "TAGame.PRI_TA:ClientStayAsPartyLeader")
			if pri.maxStreamID != 107 {
				t.Fatalf("PRI maximum stream ID: got %d", pri.maxStreamID)
			}

			aliases := map[string]string{
				"Archetypes.Ball.Ball_Default":                     "TAGame.Ball_TA",
				"Archetypes.Car.Car_Default":                       "TAGame.Car_TA",
				"Archetypes.CarComponents.CarComponent_Boost":      "TAGame.CarComponent_Boost_TA",
				"Archetypes.CarComponents.CarComponent_Dodge":      "TAGame.CarComponent_Dodge_TA",
				"Archetypes.CarComponents.CarComponent_DoubleJump": "TAGame.CarComponent_DoubleJump_TA",
				"Archetypes.CarComponents.CarComponent_FlipCar":    "TAGame.CarComponent_FlipCar_TA",
				"Archetypes.CarComponents.CarComponent_Jump":       "TAGame.CarComponent_Jump_TA",
				"Archetypes.GameEvent.GameEvent_Soccar":            "TAGame.GameEvent_Soccar_TA",
				"Archetypes.Teams.Team0":                           "TAGame.Team_Soccar_TA",
				"Archetypes.Teams.Team1":                           "TAGame.Team_Soccar_TA",
				"ProjectX.Default__NetModeReplicator_X":            "ProjectX.NetModeReplicator_X",
				"TAGame.Default__PRI_TA":                           "TAGame.PRI_TA",
				"TAGame.Default__CameraSettingsActor_TA":           "TAGame.CameraSettingsActor_TA",
			}
			for objectName, className := range aliases {
				got := fixtureAttributes(t, replay, lookup, objectName)
				if got.classObjectIndex != fixtureObjectIndex(t, replay, className) {
					t.Errorf("%s resolved to %s, want %s", objectName, replay.Objects[got.classObjectIndex], className)
				}
			}
			var pickup string
			for _, name := range replay.Objects {
				if strings.Contains(name, ":PersistentLevel.VehiclePickup_Boost_TA_") {
					pickup = name
					break
				}
			}
			if pickup == "" {
				t.Fatal("fixture has no boost pickup instance")
			}
			got := fixtureAttributes(t, replay, lookup, pickup)
			if got.classObjectIndex != fixtureObjectIndex(t, replay, "TAGame.VehiclePickup_Boost_TA") {
				t.Fatalf("%s resolved to %s", pickup, replay.Objects[got.classObjectIndex])
			}
		})
	}
}

func TestAttributeLookupUnambiguousParentAndOverrides(t *testing.T) {
	replay := syntheticAttributeReplay()
	lookup, err := newAttributeLookup(replay)
	if err != nil {
		t.Fatal(err)
	}
	root, err := lookup.forObject(0)
	if err != nil {
		t.Fatal(err)
	}
	child, err := lookup.forObject(1)
	if err != nil {
		t.Fatal(err)
	}
	if root.streams[5] != 2 || root.maxStreamID != 5 {
		t.Fatalf("unexpected root attributes: %+v", root)
	}
	if child.streams[5] != 3 || child.streams[8] != 4 || child.maxStreamID != 8 {
		t.Fatalf("unexpected child attributes: %+v", child)
	}
	if _, err := lookup.forObject(5); err == nil || !strings.Contains(err.Error(), "no known class") {
		t.Fatalf("unknown archetype: got %v", err)
	}
	for _, index := range []int32{-1, int32(len(replay.Objects))} {
		if _, err := lookup.forObject(index); err == nil || !strings.Contains(err.Error(), "out of range") {
			t.Fatalf("object %d: got %v", index, err)
		}
	}
}

func TestAttributeLookupRejectsInvalidHierarchy(t *testing.T) {
	tests := []struct {
		name   string
		replay *rocket.Replay
		want   string
	}{
		{name: "nil replay", want: "replay is nil"},
		{
			name: "missing parent",
			replay: &rocket.Replay{
				Objects:  []string{"Game.Child"},
				NetCache: []*rocket.ClassNetCache{{ObjectIndex: 0, CacheID: 1, ParentID: 99}},
			},
			want: "missing parent cache ID 99",
		},
		{
			name: "ambiguous parent",
			replay: &rocket.Replay{
				Objects: []string{"Game.Root", "Game.A", "Game.B", "Game.Child"},
				NetCache: []*rocket.ClassNetCache{
					{ObjectIndex: 0, CacheID: 0, ParentID: 0},
					{ObjectIndex: 1, CacheID: 1, ParentID: 0},
					{ObjectIndex: 2, CacheID: 1, ParentID: 0},
					{ObjectIndex: 3, CacheID: 2, ParentID: 1},
				},
			},
			want: "ambiguous parent cache ID 1",
		},
		{
			name: "cycle",
			replay: &rocket.Replay{
				Objects: []string{"Game.A", "Game.B"},
				NetCache: []*rocket.ClassNetCache{
					{ObjectIndex: 0, CacheID: 1, ParentID: 2},
					{ObjectIndex: 1, CacheID: 2, ParentID: 1},
				},
			},
			want: "parent cycle",
		},
		{
			name: "missing known parent",
			replay: &rocket.Replay{
				Objects:  []string{"Engine.Actor"},
				NetCache: []*rocket.ClassNetCache{{ObjectIndex: 0, CacheID: 20}},
			},
			want: "missing parent class",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := newAttributeLookup(tt.replay); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestAttributeLookupRejectsInvalidIndexes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*rocket.Replay)
		want   string
	}{
		{"class index", func(r *rocket.Replay) { r.ClassIndices[0].Index = -1 }, "out of range"},
		{"class name", func(r *rocket.Replay) { r.ClassIndices[0].Class = "Game.Other" }, "names"},
		{"cache index", func(r *rocket.Replay) { r.NetCache[0].ObjectIndex = int32(len(r.Objects)) }, "out of range"},
		{"property index", func(r *rocket.Replay) { r.NetCache[0].Properties[0].ObjectIndex = -1 }, "out of range"},
		{"negative stream", func(r *rocket.Replay) { r.NetCache[0].Properties[0].StreamID = -1 }, "negative stream ID"},
		{"duplicate stream", func(r *rocket.Replay) {
			r.NetCache[0].Properties = append(r.NetCache[0].Properties, &rocket.NetCacheProperty{ObjectIndex: 3, StreamID: 5})
		}, "duplicate stream ID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replay := syntheticAttributeReplay()
			tt.mutate(replay)
			if _, err := newAttributeLookup(replay); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestAttributeLookupRejectsAmbiguousInstanceClass(t *testing.T) {
	replay := &rocket.Replay{
		Objects: []string{"Core.Object", "Game.Root", "Other.Root", "Map.TheWorld:PersistentLevel.Root_0"},
		NetCache: []*rocket.ClassNetCache{
			{ObjectIndex: 0, CacheID: 0, ParentID: 0},
			{ObjectIndex: 1, CacheID: 1, ParentID: 0},
			{ObjectIndex: 2, CacheID: 2, ParentID: 0},
		},
	}
	lookup, err := newAttributeLookup(replay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lookup.forObject(3); err == nil || !strings.Contains(err.Error(), "ambiguous class") {
		t.Fatalf("got %v, want ambiguous class", err)
	}
}

func syntheticAttributeReplay() *rocket.Replay {
	return &rocket.Replay{
		Objects: []string{"Game.Root", "Game.Child", "Game.Root:Base", "Game.Child:Override", "Game.Child:Own", "Archetypes.Unknown.Thing"},
		ClassIndices: []*rocket.ClassIndex{
			{Class: "Game.Root", Index: 0},
			{Class: "Game.Child", Index: 1},
		},
		NetCache: []*rocket.ClassNetCache{
			{ObjectIndex: 0, CacheID: 0, ParentID: 0, Properties: []*rocket.NetCacheProperty{{ObjectIndex: 2, StreamID: 5}}},
			{ObjectIndex: 1, CacheID: 1, ParentID: 0, Properties: []*rocket.NetCacheProperty{{ObjectIndex: 3, StreamID: 5}, {ObjectIndex: 4, StreamID: 8}}},
		},
	}
}

func fixtureObjectIndex(t *testing.T, replay *rocket.Replay, name string) int32 {
	t.Helper()
	for index, object := range replay.Objects {
		if object == name {
			return int32(index)
		}
	}
	t.Fatalf("fixture object %q not found", name)
	return -1
}

func fixtureAttributes(t *testing.T, replay *rocket.Replay, lookup *attributeLookup, name string) *classAttributes {
	t.Helper()
	attributes, err := lookup.forObject(fixtureObjectIndex(t, replay, name))
	if err != nil {
		t.Fatal(err)
	}
	return attributes
}

func assertAttribute(t *testing.T, replay *rocket.Replay, attributes *classAttributes, stream int32, name string) {
	t.Helper()
	index, ok := attributes.streams[stream]
	if !ok {
		t.Fatalf("stream %d is missing", stream)
	}
	if replay.Objects[index] != name {
		t.Fatalf("stream %d: got %q, want %q", stream, replay.Objects[index], name)
	}
}
