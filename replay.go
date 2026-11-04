package rocket

// Replay contains the replay data
type Replay struct {
	HeaderInfo   *ReplayHeader
	Version      string
	Properties   map[string]*Property
	Levels       []string
	Keyframes    []*Keyframe
	NetworkData  []byte // Raw, undecoded network stream.
	Frames       []NetworkFrame
	DebugData    []*DebugData
	Tickmarks    []*Tickmark
	Packages     []string
	Objects      []string
	Names        []string
	ClassIndices []*ClassIndex
	NetCache     []*ClassNetCache
}

type Keyframe struct {
	Time     float32
	Frame    int32
	Position int32
}

type DebugData struct {
	Frame int32
	User  string
	Text  string
}

type Tickmark struct {
	Frame       int32
	Description string
}

// NetworkFrame contains actor replications in their original stream order.
type NetworkFrame struct {
	Time   float32
	Delta  float32
	Actors []ActorReplication
}

// ActorReplication describes one spawn, deletion, or set of attribute updates.
// Actor IDs can be reused after deletion.
type ActorReplication struct {
	ActorID    int32
	Spawn      *SpawnedActor
	Deleted    bool
	Attributes []AttributeUpdate
}

// SpawnedActor identifies a new actor and any initial position and rotation.
// NameIndex is absent in replay versions that do not encode an actor name.
type SpawnedActor struct {
	NameIndex   *int32
	ObjectIndex int32
	Location    *Vector3i
	Rotation    *Rotation
}

// AttributeUpdate identifies an attribute in Replay.Objects and its decoded value.
type AttributeUpdate struct {
	StreamID    int32
	ObjectIndex int32
	Value       interface{}
}

// Vector3i is an integer position in the network stream.
type Vector3i struct {
	X, Y, Z int32
}

// Rotation contains the optional byte-packed rotation axes.
type Rotation struct {
	Yaw, Pitch, Roll *int8
}

// ActiveActor identifies an actor reference and its active flag.
type ActiveActor struct {
	Active  bool
	ActorID int32
}

// ClassIndex maps a class name to its index in Replay.Objects.
type ClassIndex struct {
	Class string
	Index int32
}

// ClassNetCache describes a class and its replicated properties.
type ClassNetCache struct {
	ObjectIndex int32 // Index into Replay.Objects.
	ParentID    int32 // CacheID of the parent class.
	CacheID     int32
	Properties  []*NetCacheProperty
}

// NetCacheProperty maps an object index to a network stream ID.
type NetCacheProperty struct {
	ObjectIndex int32
	StreamID    int32
}

// ReplayHeader contains various details from the header portion of the replay file
type ReplayHeader struct {
	HeaderSize      int32
	CRC             uint32
	EngineVersion   int
	LicenseeVersion int
	NetVersion      int
	ClassName       string
}

// NewReplay creates a new instance of a replay
func NewReplay() *Replay {
	return &Replay{}
}
