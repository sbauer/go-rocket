package rocket

// Replay contains the replay data
type Replay struct {
	HeaderInfo *ReplayHeader
	Version    string
	Properties map[string]*Property
}

// ReplayHeader contains various details from the header portion of the replay file
type ReplayHeader struct {
	HeaderSize      uint32
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
