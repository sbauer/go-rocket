package rocket

// Vector3f contains packed network-vector components divided by 100.
type Vector3f struct {
	X, Y, Z float32
}

// Quaternion represents a replicated orientation. Before network version 7,
// only X, Y, and Z are encoded; W is zero.
type Quaternion struct {
	X, Y, Z, W float32
}

// RigidBodyState contains a position, orientation, and optional velocities.
type RigidBodyState struct {
	Sleeping        bool
	Location        Vector3f
	Rotation        Quaternion
	LinearVelocity  *Vector3f
	AngularVelocity *Vector3f
}

// CameraSettings contains a player's replicated camera configuration.
type CameraSettings struct {
	FOV, Height, Angle, Distance, Stiffness, Swivel float32
	Transition                                      *float32
	Acceleration, Deceleration, FreeLookSpeed       *float32
	UnconstrainedRotation, FreeLookSmoothing        *bool
}

// Demolition describes the actors and velocities involved in a demolition.
type Demolition struct {
	Attacker, Victim                 ActiveActor
	AttackerVelocity, VictimVelocity Vector3f
}

// DemolitionExtended includes additional actor references for a demolition.
type DemolitionExtended struct {
	AttackerPRI, SelfDemo, GoalExplosionOwner ActiveActor
	SelfDemolish                              bool
	Attacker, Victim                          ActiveActor
	AttackerVelocity, VictimVelocity          Vector3f
}

// DemolitionFX includes a custom demolition effect.
type DemolitionFX struct {
	CustomFX   ActiveActor
	Demolition Demolition
}

// Explosion describes the source and location of an explosion.
type Explosion struct {
	Source   ActiveActor
	Location Vector3f
}

// ExtendedExplosion includes a secondary actor reference.
type ExtendedExplosion struct {
	Explosion      Explosion
	SecondaryActor ActiveActor
}

// Pickup describes an older pickup update.
type Pickup struct {
	Instigator *int32
	PickedUp   bool
}

// PickupNew describes a pickup update using a byte-valued state.
type PickupNew struct {
	Instigator *int32
	PickedUp   byte
}

// FlaggedByte contains a flag followed by a byte.
type FlaggedByte struct {
	Flag  bool
	Value byte
}

// UniqueID identifies a local player and their platform-specific remote ID.
// RemoteID is uint32 for split-screen, uint64 for Steam/Xbox/QQ, string for
// Epic, or a PlayStationID, SwitchID, or PsyNetID for those platforms.
// Party leader updates contain a *UniqueID, which is nil when no leader is set.
type UniqueID struct {
	SystemID byte
	RemoteID interface{}
	LocalID  byte
}

type PlayStationID struct {
	Name     string
	OnlineID uint64
	Unknown  []byte
}

type SwitchID struct {
	OnlineID uint64
	Unknown  []byte
}

type PsyNetID struct {
	OnlineID uint64
	Unknown  []byte
}

// Reservation describes a player's slot in a match.
type Reservation struct {
	Number   byte
	UniqueID UniqueID
	Name     *string
	Flag1    bool
	Flag2    bool
	Extra    *byte
}

type HonorDuelChallenge struct {
	Challenger, Defender UniqueID
}

// Loadout contains the product IDs replicated for one team.
type Loadout struct {
	Version                                             byte
	Body, Decal, Wheels, RocketTrail, Antenna, Topper   uint32
	Unknown1                                            uint32
	Unknown2, EngineAudio, Trail, GoalExplosion, Banner *uint32
	ProductID                                           *uint32
	Extra                                               []uint32
}

type TeamLoadout struct {
	Blue, Orange Loadout
}

// ProductAttribute describes an online loadout attribute. Value is nil for
// an absent color, uint32 for numeric attributes, string for titles, or
// ProductCertification for certified items.
type ProductAttribute struct {
	Flag        bool
	ObjectIndex int32
	Value       interface{}
}

type ProductCertification struct {
	Stat, Value uint32
}

type OnlineLoadout [][]ProductAttribute

type TeamOnlineLoadouts struct {
	Blue, Orange OnlineLoadout
	Flag1, Flag2 bool
}

type TeamPaint struct {
	Team, PrimaryColor, AccentColor byte
	PrimaryFinish, AccentFinish     uint32
}

type StatEvent struct {
	Flag        bool
	ObjectIndex int32
}

type ClubColors struct {
	BlueFlag, OrangeFlag   bool
	BlueColor, OrangeColor byte
}

type PrivateMatchSettings struct {
	Mutators   string
	JoinableBy uint32
	MaxPlayers uint32
	GameName   string
	Password   string
	Flag       bool
}

type MusicStinger struct {
	Flag    bool
	Cue     uint32
	Trigger byte
}

type ReplicatedBoost struct {
	GrantCount, BoostAmount, Unknown1, Unknown2 byte
}

type PickupInfo struct {
	Available [3]ActiveActor
	Preview   bool
}

type GameMode struct {
	Width byte
	Value byte
}

type ReplicatedTitle struct {
	Flag1, Flag2 bool
	Values       [5]uint32
	Flag3        bool
}

type ReplicatedStatTitle struct {
	Flag1, Flag2 bool
	Name         string
	Index, Value uint32
}

type WeldedInfo struct {
	Actor    ActiveActor
	Offset   Vector3f
	Mass     float32
	Rotation Rotation
}

type AppliedDamage struct {
	ID                       byte
	Position                 Vector3f
	DamageIndex, TotalDamage int32
}

type DamageState struct {
	TileState       byte
	Damaged         bool
	Offender        int32
	BallPosition    Vector3f
	DirectHit, Flag bool
}

type Impulse struct {
	CompressedRotation int32
	Speed              float32
}

type LogoData struct {
	LogoID     uint32
	SwapColors bool
}
