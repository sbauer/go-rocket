package parsers

import (
	"bytes"
	"errors"
	"io"
	"math"
	"math/bits"
	"reflect"
	"strings"
	"testing"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

type networkTestWriter struct {
	data []byte
	bits int
}

func (writer *networkTestWriter) write(value uint64, width int) {
	for i := 0; i < width; i++ {
		if writer.bits%8 == 0 {
			writer.data = append(writer.data, 0)
		}
		if value&(uint64(1)<<uint(i)) != 0 {
			writer.data[writer.bits/8] |= 1 << uint(writer.bits%8)
		}
		writer.bits++
	}
}

func (writer *networkTestWriter) bounded(value, max uint64) {
	width := bits.Len64(max) - 1
	low := value & ((uint64(1) << uint(width)) - 1)
	writer.write(low, width)
	high := uint64(1) << uint(width)
	if high < max-low {
		writer.write(value>>uint(width), 1)
	}
}

func (writer *networkTestWriter) frame(time, delta float32) {
	writer.write(uint64(math.Float32bits(time)), 32)
	writer.write(uint64(math.Float32bits(delta)), 32)
}

func (writer *networkTestWriter) actor(id uint64, alive, isNew bool) {
	writer.write(1, 1)
	writer.bounded(id, 8)
	if !alive {
		writer.write(0, 1)
		return
	}
	writer.write(1, 1)
	if isNew {
		writer.write(1, 1)
	} else {
		writer.write(0, 1)
	}
}

func (writer *networkTestWriter) spawn(name, object uint32) {
	writer.write(uint64(name), 32)
	writer.write(0, 1)
	writer.write(uint64(object), 32)
	writer.bounded(0, 22)
	writer.write(3, 2)
	writer.write(0, 2)
	writer.write(2, 2)
}

func networkTestReplay() *rocket.Replay {
	return &rocket.Replay{
		HeaderInfo: &rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: 10},
		Properties: map[string]*rocket.Property{
			"NumFrames":   {Value: int32(3)},
			"MaxChannels": {Value: int32(8)},
		},
		Objects: []string{"Core.Object", "Engine.Actor", "Engine.Actor:bHidden", "Engine.Actor:DrawScale"},
		ClassIndices: []*rocket.ClassIndex{
			{Class: "Core.Object", Index: 0},
			{Class: "Engine.Actor", Index: 1},
		},
		NetCache: []*rocket.ClassNetCache{
			{ObjectIndex: 0, CacheID: 0, ParentID: 0},
			{ObjectIndex: 1, CacheID: 2, ParentID: 0, Properties: []*rocket.NetCacheProperty{
				{ObjectIndex: 2, StreamID: 0},
				{ObjectIndex: 3, StreamID: 1},
			}},
		},
	}
}

func networkTestData() []byte {
	writer := &networkTestWriter{}
	writer.frame(1, 0.1)
	writer.actor(3, true, true)
	writer.spawn(5, 1)
	writer.write(0, 1)

	writer.frame(1.1, 0.1)
	writer.actor(3, true, false)
	writer.write(1, 1)
	writer.bounded(0, 2)
	writer.write(1, 1)
	writer.write(1, 1)
	writer.bounded(1, 2)
	writer.write(uint64(math.Float32bits(1.5)), 32)
	writer.write(0, 1)
	writer.actor(3, false, false)
	writer.actor(3, true, true)
	writer.spawn(6, 1)
	writer.write(0, 1)

	writer.frame(1.2, 0.1)
	writer.actor(3, true, false)
	writer.write(1, 1)
	writer.bounded(0, 2)
	writer.write(0, 1)
	writer.write(0, 1)
	writer.write(0, 1)
	return writer.data
}

func TestDecodeNetworkActorLifecycle(t *testing.T) {
	replay := networkTestReplay()
	replay.NetworkData = networkTestData()
	frames, err := decodeNetwork(replay)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 || frames[0].Time != 1 || frames[1].Time != 1.1 {
		t.Fatalf("unexpected frames: %+v", frames)
	}
	if len(frames[0].Actors) != 1 || len(frames[1].Actors) != 3 {
		t.Fatalf("unexpected actor counts: %d, %d", len(frames[0].Actors), len(frames[1].Actors))
	}
	first := frames[0].Actors[0]
	if first.ActorID != 3 || first.Spawn == nil || first.Spawn.NameIndex == nil ||
		*first.Spawn.NameIndex != 5 || first.Spawn.ObjectIndex != 1 ||
		!reflect.DeepEqual(first.Spawn.Location, &rocket.Vector3i{X: 1, Y: -2}) ||
		first.Spawn.Rotation != nil {
		t.Fatalf("unexpected spawn: %+v", first)
	}
	updates := frames[1].Actors[0].Attributes
	if len(updates) != 2 || updates[0].ObjectIndex != 2 || updates[0].Value != true ||
		updates[1].ObjectIndex != 3 || updates[1].Value != float32(1.5) {
		t.Fatalf("unexpected updates: %+v", updates)
	}
	if !frames[1].Actors[1].Deleted || frames[1].Actors[2].Spawn == nil ||
		*frames[1].Actors[2].Spawn.NameIndex != 6 {
		t.Fatalf("deletion and reuse were not preserved: %+v", frames[1].Actors)
	}
	if len(frames[2].Actors) != 1 || len(frames[2].Actors[0].Attributes) != 1 ||
		frames[2].Actors[0].Attributes[0].Value != false {
		t.Fatalf("reused actor could not be updated: %+v", frames[2].Actors)
	}
}

func TestDecodeNetworkReplaySpawn(t *testing.T) {
	replay, err := Parse(source.FromFile("../samples/first.replay"))
	if err != nil {
		t.Fatal(err)
	}
	lookup, err := newAttributeLookup(replay)
	if err != nil {
		t.Fatal(err)
	}
	reader := newBitReader(replay.NetworkData)
	if _, err := reader.readFloat32(); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.readFloat32(); err != nil {
		t.Fatal(err)
	}
	present, err := reader.readBit()
	if err != nil || !present {
		t.Fatalf("first actor presence: %v, %v", present, err)
	}
	maxChannels, err := networkIntProperty(replay, "MaxChannels", 1023)
	if err != nil {
		t.Fatal(err)
	}
	decoder := &networkDecoder{
		reader: reader, replay: replay, lookup: lookup,
		actors: make(map[int32]*classAttributes), maxChannels: uint64(maxChannels),
	}
	actor, err := decoder.readActor()
	if err != nil {
		t.Fatal(err)
	}
	if actor.ActorID != 0 || actor.Spawn == nil || actor.Spawn.NameIndex == nil ||
		*actor.Spawn.NameIndex != 0 ||
		replay.Objects[actor.Spawn.ObjectIndex] != "GameInfo_Soccar.GameInfo.GameInfo_Soccar:GameReplicationInfoArchetype" ||
		!reflect.DeepEqual(actor.Spawn.Location, &rocket.Vector3i{}) {
		t.Fatalf("unexpected first replay actor: %+v", actor)
	}
}

func TestDecodeNetworkSpawnVersions(t *testing.T) {
	tests := []struct {
		name      string
		engine    int
		licensee  int
		matchType string
		hasName   bool
	}{
		{"old", 868, 13, "", false},
		{"online", 868, 14, "Online", true},
		{"old LAN", 868, 19, "Lan", false},
		{"new LAN", 868, 20, "Lan", true},
		{"new engine", 869, 0, "Lan", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replay := networkTestReplay()
			replay.HeaderInfo.EngineVersion = tt.engine
			replay.HeaderInfo.LicenseeVersion = tt.licensee
			replay.Properties["MatchType"] = &rocket.Property{Value: tt.matchType}
			lookup, err := newAttributeLookup(replay)
			if err != nil {
				t.Fatal(err)
			}
			writer := &networkTestWriter{}
			if tt.hasName {
				writer.write(17, 32)
			}
			writer.write(0, 1)
			writer.write(1, 32)
			writer.bounded(0, 22)
			writer.write(2, 2)
			writer.write(2, 2)
			writer.write(2, 2)
			decoder := &networkDecoder{reader: newBitReader(writer.data), replay: replay, lookup: lookup}
			spawn, _, err := decoder.readSpawn()
			if err != nil {
				t.Fatal(err)
			}
			if (spawn.NameIndex != nil) != tt.hasName ||
				(spawn.NameIndex != nil && *spawn.NameIndex != 17) ||
				!reflect.DeepEqual(spawn.Location, &rocket.Vector3i{}) {
				t.Fatalf("unexpected spawn: %+v", spawn)
			}
		})
	}
}

func TestSpawnTrajectoryClasses(t *testing.T) {
	tests := []struct {
		class    string
		location bool
		rotation bool
	}{
		{"TAGame.Car_TA", true, true},
		{"TAGame.GameEvent_Soccar_TA", true, false},
		{"TAGame.CrowdActor_TA", false, false},
		{"TAGame.VehiclePickup_Boost_TA", false, false},
	}
	for _, tt := range tests {
		location, rotation, err := spawnTrajectory(tt.class)
		if err != nil || location != tt.location || rotation != tt.rotation {
			t.Fatalf("%s: got %v, %v, %v", tt.class, location, rotation, err)
		}
	}
	if _, _, err := spawnTrajectory("Unknown.Class"); err == nil {
		t.Fatal("expected an error for an unknown class")
	}
}

func TestDecodeNetworkErrors(t *testing.T) {
	t.Run("unknown actor", func(t *testing.T) {
		writer := &networkTestWriter{}
		writer.frame(1, 0)
		writer.actor(2, true, false)
		writer.write(0, 1)
		writer.write(0, 1)
		replay := networkTestReplay()
		replay.Properties["NumFrames"].Value = int32(1)
		replay.NetworkData = writer.data
		_, err := decodeNetwork(replay)
		if err == nil || !strings.Contains(err.Error(), "actor 2 updated before spawn") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("truncated actor", func(t *testing.T) {
		writer := &networkTestWriter{}
		writer.frame(1, 0)
		writer.actor(2, true, true)
		replay := networkTestReplay()
		replay.Properties["NumFrames"].Value = int32(1)
		replay.NetworkData = writer.data
		_, err := decodeNetwork(replay)
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("got %v, want unexpected EOF", err)
		}
	})
	t.Run("unsupported attribute", func(t *testing.T) {
		replay := networkTestReplay()
		replay.NetworkData = networkTestData()
		replay.Objects[2] = "Engine.Actor:Unsupported"
		_, err := decodeNetwork(replay)
		if err == nil || !strings.Contains(err.Error(), `unsupported attribute "Engine.Actor:Unsupported"`) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("missing stream", func(t *testing.T) {
		writer := &networkTestWriter{}
		writer.frame(1, 0)
		writer.actor(3, true, true)
		writer.spawn(1, 1)
		writer.actor(3, true, false)
		writer.write(1, 1)
		writer.bounded(1, 3)
		replay := networkTestReplay()
		replay.Properties["NumFrames"].Value = int32(1)
		replay.NetworkData = writer.data
		replay.NetCache[1].Properties[1].StreamID = 2
		_, err := decodeNetwork(replay)
		if err == nil || !strings.Contains(err.Error(), "unknown stream ID 1") {
			t.Fatalf("got %v", err)
		}
	})
	for _, tt := range []struct {
		name  string
		value interface{}
		want  string
	}{
		{"NumFrames", int32(-1), "NumFrames"},
		{"NumFrames", "2", "not an integer"},
		{"MaxChannels", int32(0), "MaxChannels"},
	} {
		t.Run(tt.name+" "+tt.want, func(t *testing.T) {
			replay := networkTestReplay()
			replay.NetworkData = networkTestData()
			replay.Properties[tt.name].Value = tt.value
			_, err := decodeNetwork(replay)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseNetworkOption(t *testing.T) {
	data := encodeNetworkTestReplay(t, networkTestData())
	metadata, err := Parse(source.FromBytes(data))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Frames != nil || !bytes.Equal(metadata.NetworkData, networkTestData()) {
		t.Fatal("default parsing should retain raw network data without decoding frames")
	}
	parsed, err := ParseWithOptions(source.FromBytes(data), ParseOptions{DecodeNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Frames) != 3 || len(parsed.Frames[1].Actors) != 3 {
		t.Fatalf("unexpected decoded frames: %+v", parsed.Frames)
	}
}

func TestDecodeNetworkWithoutFrameCount(t *testing.T) {
	t.Run("empty stream", func(t *testing.T) {
		replay := networkTestReplay()
		delete(replay.Properties, "NumFrames")
		replay.NetworkData = nil
		frames, err := decodeNetwork(replay)
		if err != nil {
			t.Fatal(err)
		}
		if frames == nil || len(frames) != 0 {
			t.Fatalf("got %#v, want an empty frame slice", frames)
		}
	})
	t.Run("non-empty stream", func(t *testing.T) {
		replay := networkTestReplay()
		delete(replay.Properties, "NumFrames")
		replay.NetworkData = networkTestData()
		if _, err := decodeNetwork(replay); err == nil || !strings.Contains(err.Error(), "missing or negative NumFrames") {
			t.Fatalf("got %v, want missing frame count error", err)
		}
	})
}

func encodeNetworkTestReplay(t *testing.T, network []byte) []byte {
	t.Helper()
	replay := networkTestReplay()
	var header bytes.Buffer
	for _, value := range []uint32{868, 28, 10} {
		writeValue(t, &header, value)
	}
	writeString(t, &header, "Replay")
	for _, name := range []string{"NumFrames", "MaxChannels"} {
		writeString(t, &header, name)
		writeString(t, &header, "IntProperty")
		writeValue(t, &header, uint32(4))
		writeValue(t, &header, uint32(0))
		writeValue(t, &header, replay.Properties[name].Value)
	}
	writeString(t, &header, "None")

	var body bytes.Buffer
	writeValue(t, &body, int32(0))
	writeValue(t, &body, int32(0))
	writeValue(t, &body, int32(len(network)))
	body.Write(network)
	for i := 0; i < 3; i++ {
		writeValue(t, &body, int32(0))
	}
	writeValue(t, &body, int32(len(replay.Objects)))
	for _, name := range replay.Objects {
		writeString(t, &body, name)
	}
	writeValue(t, &body, int32(0))
	writeValue(t, &body, int32(len(replay.ClassIndices)))
	for _, class := range replay.ClassIndices {
		writeString(t, &body, class.Class)
		writeValue(t, &body, class.Index)
	}
	writeValue(t, &body, int32(len(replay.NetCache)))
	for _, class := range replay.NetCache {
		writeValue(t, &body, class.ObjectIndex)
		writeValue(t, &body, class.ParentID)
		writeValue(t, &body, class.CacheID)
		writeValue(t, &body, int32(len(class.Properties)))
		for _, property := range class.Properties {
			writeValue(t, &body, property.ObjectIndex)
			writeValue(t, &body, property.StreamID)
		}
	}
	var data bytes.Buffer
	writeValue(t, &data, int32(header.Len()))
	writeValue(t, &data, replayCRC32(header.Bytes()))
	data.Write(header.Bytes())
	writeValue(t, &data, int32(body.Len()))
	writeValue(t, &data, replayCRC32(body.Bytes()))
	data.Write(body.Bytes())
	return data.Bytes()
}
