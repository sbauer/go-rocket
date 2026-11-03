package rocket

// Replay contains the replay data
type Replay struct {
	HeaderInfo   *ReplayHeader
	Version      string
	Properties   map[string]*Property
	Levels       []string
	Keyframes    []*Keyframe
	NetworkData  []byte // Raw, undecoded network stream.
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
