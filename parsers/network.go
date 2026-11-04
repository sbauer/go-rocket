package parsers

import (
	"fmt"
	"math"

	"github.com/sbauer/go-rocket"
)

type networkDecoder struct {
	reader      *bitReader
	replay      *rocket.Replay
	lookup      *attributeLookup
	actors      map[int32]*classAttributes
	maxChannels uint64
}

func decodeNetwork(replay *rocket.Replay) ([]rocket.NetworkFrame, error) {
	if replay.HeaderInfo == nil {
		return nil, fmt.Errorf("missing replay header")
	}
	numFrames, err := networkIntProperty(replay, "NumFrames", -1)
	if err != nil {
		return nil, err
	}
	if numFrames < 0 {
		return nil, fmt.Errorf("missing or negative NumFrames")
	}
	maxChannels, err := networkIntProperty(replay, "MaxChannels", 1023)
	if err != nil {
		return nil, err
	}
	if maxChannels <= 0 {
		return nil, fmt.Errorf("invalid MaxChannels %d", maxChannels)
	}
	if uint64(numFrames) > uint64(len(replay.NetworkData))*8/65 {
		return nil, fmt.Errorf("NumFrames %d exceeds network data length", numFrames)
	}
	if numFrames == 0 {
		return []rocket.NetworkFrame{}, nil
	}
	lookup, err := newAttributeLookup(replay)
	if err != nil {
		return nil, fmt.Errorf("attribute lookup: %w", err)
	}
	decoder := &networkDecoder{
		reader:      newBitReader(replay.NetworkData),
		replay:      replay,
		lookup:      lookup,
		actors:      make(map[int32]*classAttributes),
		maxChannels: uint64(maxChannels),
	}
	frames := make([]rocket.NetworkFrame, 0)
	for index := int32(0); index < numFrames; index++ {
		frame, err := decoder.readFrame()
		if err != nil {
			return nil, fmt.Errorf("frame %d: %w", index, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

func networkIntProperty(replay *rocket.Replay, name string, fallback int32) (int32, error) {
	property, ok := replay.Properties[name]
	if !ok {
		return fallback, nil
	}
	if property == nil {
		return 0, fmt.Errorf("%s property is nil", name)
	}
	value, ok := property.Value.(int32)
	if !ok {
		return 0, fmt.Errorf("%s property is not an integer", name)
	}
	return value, nil
}

func (decoder *networkDecoder) readFrame() (rocket.NetworkFrame, error) {
	time, err := decoder.reader.readFloat32()
	if err != nil {
		return rocket.NetworkFrame{}, fmt.Errorf("time: %w", err)
	}
	delta, err := decoder.reader.readFloat32()
	if err != nil {
		return rocket.NetworkFrame{}, fmt.Errorf("delta: %w", err)
	}
	if math.IsNaN(float64(time)) || math.IsInf(float64(time), 0) || time < 0 {
		return rocket.NetworkFrame{}, fmt.Errorf("invalid time %v", time)
	}
	if math.IsNaN(float64(delta)) || math.IsInf(float64(delta), 0) || delta < 0 {
		return rocket.NetworkFrame{}, fmt.Errorf("invalid delta %v", delta)
	}
	if time == 0 && delta == 0 {
		return rocket.NetworkFrame{}, fmt.Errorf("end of network data before expected frame count")
	}

	frame := rocket.NetworkFrame{Time: time, Delta: delta}
	for {
		present, err := decoder.reader.readBit()
		if err != nil {
			return rocket.NetworkFrame{}, fmt.Errorf("actor presence: %w", err)
		}
		if !present {
			return frame, nil
		}
		replication, err := decoder.readActor()
		if err != nil {
			return rocket.NetworkFrame{}, err
		}
		frame.Actors = append(frame.Actors, replication)
	}
}

func (decoder *networkDecoder) readActor() (rocket.ActorReplication, error) {
	id, err := decoder.reader.readBoundedUint(decoder.maxChannels)
	if err != nil {
		return rocket.ActorReplication{}, fmt.Errorf("actor ID: %w", err)
	}
	actorID := int32(id)
	alive, err := decoder.reader.readBit()
	if err != nil {
		return rocket.ActorReplication{}, fmt.Errorf("actor %d alive: %w", actorID, err)
	}
	replication := rocket.ActorReplication{ActorID: actorID}
	if !alive {
		delete(decoder.actors, actorID)
		replication.Deleted = true
		return replication, nil
	}
	isNew, err := decoder.reader.readBit()
	if err != nil {
		return rocket.ActorReplication{}, fmt.Errorf("actor %d new: %w", actorID, err)
	}
	if isNew {
		spawn, attributes, err := decoder.readSpawn()
		if err != nil {
			return rocket.ActorReplication{}, fmt.Errorf("actor %d spawn: %w", actorID, err)
		}
		decoder.actors[actorID] = attributes
		replication.Spawn = spawn
		return replication, nil
	}
	attributes, ok := decoder.actors[actorID]
	if !ok {
		return rocket.ActorReplication{}, fmt.Errorf("actor %d updated before spawn", actorID)
	}
	for {
		present, err := decoder.reader.readBit()
		if err != nil {
			return rocket.ActorReplication{}, fmt.Errorf("actor %d attribute presence: %w", actorID, err)
		}
		if !present {
			return replication, nil
		}
		if attributes.maxStreamID < 0 {
			return rocket.ActorReplication{}, fmt.Errorf("actor %d has no attributes", actorID)
		}
		stream, err := decoder.reader.readBoundedUint(uint64(attributes.maxStreamID) + 1)
		if err != nil {
			return rocket.ActorReplication{}, fmt.Errorf("actor %d stream ID: %w", actorID, err)
		}
		objectIndex, ok := attributes.streams[int32(stream)]
		if !ok {
			return rocket.ActorReplication{}, fmt.Errorf("actor %d has unknown stream ID %d", actorID, stream)
		}
		name := decoder.replay.Objects[objectIndex]
		value, err := decodeAttribute(decoder.reader, name)
		if err != nil {
			return rocket.ActorReplication{}, fmt.Errorf("actor %d attribute %q: %w", actorID, name, err)
		}
		replication.Attributes = append(replication.Attributes, rocket.AttributeUpdate{
			StreamID:    int32(stream),
			ObjectIndex: objectIndex,
			Value:       value,
		})
	}
}

func (decoder *networkDecoder) readSpawn() (*rocket.SpawnedActor, *classAttributes, error) {
	spawn := &rocket.SpawnedActor{}
	header := decoder.replay.HeaderInfo
	isLAN := false
	if property := decoder.replay.Properties["MatchType"]; property != nil {
		isLAN = property.Value == "Lan"
	}
	hasName := header.EngineVersion > 868 ||
		(header.EngineVersion == 868 && (header.LicenseeVersion >= 20 ||
			(header.LicenseeVersion >= 14 && !isLAN)))
	if hasName {
		name, err := decoder.reader.readInt(32)
		if err != nil {
			return nil, nil, fmt.Errorf("name index: %w", err)
		}
		nameIndex := int32(name)
		spawn.NameIndex = &nameIndex
	}
	if _, err := decoder.reader.readBit(); err != nil {
		return nil, nil, fmt.Errorf("spawn flag: %w", err)
	}
	object, err := decoder.reader.readInt(32)
	if err != nil {
		return nil, nil, fmt.Errorf("object index: %w", err)
	}
	spawn.ObjectIndex = int32(object)
	attributes, err := decoder.lookup.forObject(spawn.ObjectIndex)
	if err != nil {
		return nil, nil, err
	}
	class := decoder.replay.Objects[attributes.classObjectIndex]
	location, rotation, err := spawnTrajectory(class)
	if err != nil {
		return nil, nil, err
	}
	if location {
		vector, err := decoder.reader.readVector3i(header.NetVersion)
		if err != nil {
			return nil, nil, fmt.Errorf("location: %w", err)
		}
		spawn.Location = &rocket.Vector3i{X: vector.X, Y: vector.Y, Z: vector.Z}
	}
	if rotation {
		value, err := decoder.reader.readRotation()
		if err != nil {
			return nil, nil, fmt.Errorf("rotation: %w", err)
		}
		spawn.Rotation = &rocket.Rotation{Yaw: value.Yaw, Pitch: value.Pitch, Roll: value.Roll}
	}
	return spawn, attributes, nil
}

func spawnTrajectory(class string) (location, rotation bool, err error) {
	for class != "" {
		switch class {
		case "TAGame.CrowdActor_TA", "TAGame.CrowdManager_TA", "TAGame.VehiclePickup_Boost_TA":
			return false, false, nil
		case "TAGame.RBActor_TA":
			return true, true, nil
		case "Engine.Actor":
			return true, false, nil
		}
		parent, ok := knownClassParents[class]
		if !ok {
			return false, false, fmt.Errorf("unknown spawn trajectory for class %q", class)
		}
		class = parent
	}
	return false, false, nil
}
