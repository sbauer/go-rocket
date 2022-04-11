package rocket

// Replay contains the replay data
type Replay struct {
	HeaderInfo *ReplayHeader
	Version    string
	Properties map[string]*Property
	Levels     []string
	Keyframes  []*Keyframe
	DebugData  []*DebugData
	Tickmarks  []*Tickmark
	Packages   []string
	Objects    []string
	Names      []string
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
