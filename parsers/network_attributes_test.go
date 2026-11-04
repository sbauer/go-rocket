package parsers

import (
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/source"
)

func TestDecodeNetworkFixtures(t *testing.T) {
	for _, tt := range []struct {
		name   string
		frames int
		bodies int
	}{
		{"first", 1477, 3814},
		{"second", 6107, 24595},
		{"third", 11314, 37989},
	} {
		t.Run(tt.name, func(t *testing.T) {
			replay, err := ParseWithOptions(
				source.FromFile(fmt.Sprintf("../samples/%s.replay", tt.name)),
				ParseOptions{DecodeNetwork: true},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(replay.Frames) != tt.frames {
				t.Fatalf("got %d frames, want %d", len(replay.Frames), tt.frames)
			}
			var bodies int
			for _, frame := range replay.Frames {
				for _, actor := range frame.Actors {
					for _, attribute := range actor.Attributes {
						if body, ok := attribute.Value.(rocket.RigidBodyState); ok {
							if bodies == 0 && (!body.Sleeping ||
								body.Location != (rocket.Vector3f{Z: 92.74}) ||
								math.Abs(float64(body.Rotation.W-1)) > 1e-6 ||
								body.LinearVelocity != nil || body.AngularVelocity != nil) {
								t.Fatalf("unexpected first rigid body: %+v", body)
							}
							bodies++
						}
					}
				}
			}
			if bodies != tt.bodies {
				t.Fatalf("got %d rigid body updates, want %d", bodies, tt.bodies)
			}
			if tt.name == "first" {
				assertFirstFixtureAttributes(t, replay)
			}
		})
	}
}

func assertFirstFixtureAttributes(t *testing.T, replay *rocket.Replay) {
	t.Helper()
	transition, instigator := float32(1), int32(24)
	for _, tt := range []struct {
		frame int
		name  string
		want  interface{}
	}{
		{0, "TAGame.CameraSettingsActor_TA:ProfileSettings", rocket.CameraSettings{
			FOV: 110, Height: 100, Angle: -5, Distance: 270, Stiffness: 0.5, Swivel: 6, Transition: &transition,
		}},
		{0, "TAGame.Car_TA:TeamPaint", rocket.TeamPaint{Team: 1, PrimaryColor: 33, PrimaryFinish: 270, AccentFinish: 270}},
		{128, "TAGame.VehiclePickup_TA:NewReplicatedPickupData", rocket.PickupNew{Instigator: &instigator, PickedUp: 1}},
		{149, "TAGame.CarComponent_Dodge_TA:DodgeTorque", rocket.Vector3f{X: 0.02, Y: 2.24}},
		{828, "TAGame.GameEvent_Soccar_TA:ReplicatedStatEvent", rocket.StatEvent{ObjectIndex: 354}},
		{828, "TAGame.Ball_TA:ReplicatedExplosionDataExtended", rocket.ExtendedExplosion{
			Explosion:      rocket.Explosion{Source: rocket.ActiveActor{ActorID: 355}, Location: rocket.Vector3f{X: 1.87, Y: -51.95, Z: 5.36}},
			SecondaryActor: rocket.ActiveActor{Active: true, ActorID: 2},
		}},
		{1418, "TAGame.Car_TA:ReplicatedDemolish", rocket.Demolition{
			Attacker: rocket.ActiveActor{ActorID: -1}, Victim: rocket.ActiveActor{Active: true, ActorID: 101},
			VictimVelocity: rocket.Vector3f{X: -9, Y: 20.94, Z: -3.05},
		}},
	} {
		found := false
		for _, actor := range replay.Frames[tt.frame].Actors {
			for _, attribute := range actor.Attributes {
				if replay.Objects[attribute.ObjectIndex] == tt.name && !found {
					found = true
					if !reflect.DeepEqual(attribute.Value, tt.want) {
						t.Errorf("frame %d %s: got %#v, want %#v", tt.frame, tt.name, attribute.Value, tt.want)
					}
				}
			}
		}
		if !found {
			t.Errorf("frame %d: missing %s", tt.frame, tt.name)
		}
	}
}

func attributeTestWriter() *networkTestWriter {
	writer := &networkTestWriter{}
	writer.write(5, 3)
	return writer
}

func (writer *networkTestWriter) bytes(values []byte) {
	for _, value := range values {
		writer.write(uint64(value), 8)
	}
}

func (writer *networkTestWriter) text(value string) {
	writer.write(uint64(len(value)+1), 32)
	writer.bytes([]byte(value))
	writer.write(0, 8)
}

func (writer *networkTestWriter) vector(netVersion int, x, y, z int32) {
	limit := uint64(20)
	if netVersion >= 7 {
		limit = 22
	}
	for size := uint64(0); size < limit; size++ {
		bias := int32(1) << uint(size+1)
		if x >= -bias && x < bias && y >= -bias && y < bias && z >= -bias && z < bias {
			writer.bounded(size, limit)
			for _, component := range []int32{x, y, z} {
				writer.write(uint64(component+bias), int(size)+2)
			}
			return
		}
	}
	panic("test vector out of range")
}

func attributeTestDecoder(writer *networkTestWriter, header rocket.ReplayHeader) *networkDecoder {
	reader := newBitReader(writer.data)
	_, _ = reader.readBits(3)
	return &networkDecoder{
		reader: reader,
		replay: &rocket.Replay{HeaderInfo: &header, Properties: make(map[string]*rocket.Property)},
	}
}

func assertNetworkAttribute(t *testing.T, decoder *networkDecoder, name string, want interface{}) {
	t.Helper()
	got, err := decoder.readAttribute(name)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", name, got, want)
	}
	sentinel, err := decoder.reader.readBit()
	if err != nil || !sentinel {
		t.Fatalf("%s did not consume the expected number of bits: sentinel=%v err=%v", name, sentinel, err)
	}
}

func TestNetworkUniqueIDPlatforms(t *testing.T) {
	const id = uint64(0x123456789abcdef0)
	unknown8 := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	unknown16 := append(append([]byte{}, unknown8...), unknown8...)
	unknown24 := append(append([]byte{}, unknown16...), unknown8...)
	tests := []struct {
		name       string
		system     byte
		netVersion int
		payload    func(*networkTestWriter)
		remote     interface{}
	}{
		{"split screen", 0, 10, func(w *networkTestWriter) { w.write(0x123456, 24) }, uint32(0x123456)},
		{"Steam", 1, 10, func(w *networkTestWriter) { w.write(id, 64) }, id},
		{"PlayStation old", 2, 0, func(w *networkTestWriter) {
			w.bytes(append([]byte("player"), make([]byte, 10)...))
			w.bytes(unknown8)
			w.write(id, 64)
		}, rocket.PlayStationID{Name: "player", Unknown: unknown8, OnlineID: id}},
		{"PlayStation new", 2, 1, func(w *networkTestWriter) {
			w.bytes(append([]byte("player"), make([]byte, 10)...))
			w.bytes(unknown16)
			w.write(id, 64)
		}, rocket.PlayStationID{Name: "player", Unknown: unknown16, OnlineID: id}},
		{"Xbox", 4, 10, func(w *networkTestWriter) { w.write(id, 64) }, id},
		{"QQ", 5, 10, func(w *networkTestWriter) { w.write(id, 64) }, id},
		{"Switch", 6, 10, func(w *networkTestWriter) {
			w.write(id, 64)
			w.bytes(unknown24)
		}, rocket.SwitchID{OnlineID: id, Unknown: unknown24}},
		{"PsyNet old", 7, 9, func(w *networkTestWriter) {
			w.write(id, 64)
			w.bytes(unknown24)
		}, rocket.PsyNetID{OnlineID: id, Unknown: unknown24}},
		{"PsyNet new", 7, 10, func(w *networkTestWriter) { w.write(id, 64) }, rocket.PsyNetID{OnlineID: id}},
		{"Epic", 11, 10, func(w *networkTestWriter) { w.text("epic-id") }, "epic-id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := attributeTestWriter()
			writer.write(uint64(tt.system), 8)
			tt.payload(writer)
			writer.write(3, 8)
			writer.write(1, 1)
			decoder := attributeTestDecoder(writer, rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: tt.netVersion})
			assertNetworkAttribute(t, decoder, "Engine.PlayerReplicationInfo:UniqueId",
				rocket.UniqueID{SystemID: tt.system, RemoteID: tt.remote, LocalID: 3})
		})
	}
}

func TestNetworkReservationsAndPartyLeader(t *testing.T) {
	header := rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: 10}
	t.Run("reservation with split-screen name", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(5, 3)
		writer.write(0, 8)
		writer.write(42, 24)
		writer.write(2, 8)
		writer.bytes([]byte{'g', 'u', 'e', 's', 't', 0})
		writer.write(1, 1)
		writer.write(0, 1)
		writer.write(37, 6)
		writer.write(1, 1)
		name, extra := "guest", byte(37)
		want := rocket.Reservation{
			Number: 5, UniqueID: rocket.UniqueID{RemoteID: uint32(42), LocalID: 2},
			Name: &name, Flag1: true, Extra: &extra,
		}
		assertNetworkAttribute(t, attributeTestDecoder(writer, header), "ProjectX.GRI_X:Reservations", want)
	})
	t.Run("old reservation without name or extra", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(1, 3)
		writer.write(0, 8)
		writer.write(0, 24)
		writer.write(0, 8)
		writer.write(0, 1)
		writer.write(1, 1)
		writer.write(1, 1)
		old := rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 11}
		want := rocket.Reservation{Number: 1, UniqueID: rocket.UniqueID{RemoteID: uint32(0)}, Flag2: true}
		assertNetworkAttribute(t, attributeTestDecoder(writer, old), "ProjectX.GRI_X:Reservations", want)
	})
	t.Run("party leader absent", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(0, 8)
		writer.write(1, 1)
		var want *rocket.UniqueID
		assertNetworkAttribute(t, attributeTestDecoder(writer, header), "TAGame.PRI_TA:PartyLeader", want)
	})
	t.Run("party leader present", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(4, 8)
		writer.write(123, 64)
		writer.write(1, 8)
		writer.write(1, 1)
		want := &rocket.UniqueID{SystemID: 4, RemoteID: uint64(123), LocalID: 1}
		assertNetworkAttribute(t, attributeTestDecoder(writer, header), "TAGame.PRI_TA:PartyLeader", want)
	})
}

func TestNetworkCameraVersions(t *testing.T) {
	tests := []struct {
		name                string
		header              rocket.ReplayHeader
		transition, newAxes bool
	}{
		{"old", rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 19}, false, false},
		{"transition", rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 20}, true, false},
		{"before new axes", rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 34, NetVersion: 11}, true, false},
		{"new axes", rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 34, NetVersion: 12}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := attributeTestWriter()
			for _, value := range []float32{110, 100, -5, 270, 0.5, 6} {
				writer.write(uint64(math.Float32bits(value)), 32)
			}
			want := rocket.CameraSettings{FOV: 110, Height: 100, Angle: -5, Distance: 270, Stiffness: 0.5, Swivel: 6}
			if tt.transition {
				transition := float32(1.5)
				writer.write(uint64(math.Float32bits(transition)), 32)
				want.Transition = &transition
			}
			if tt.newAxes {
				acceleration, deceleration, speed := float32(1), float32(2), float32(3)
				for _, value := range []float32{acceleration, deceleration, speed} {
					writer.write(uint64(math.Float32bits(value)), 32)
				}
				writer.write(1, 1)
				writer.write(0, 1)
				unconstrained, smoothing := true, false
				want.Acceleration, want.Deceleration, want.FreeLookSpeed = &acceleration, &deceleration, &speed
				want.UnconstrainedRotation, want.FreeLookSmoothing = &unconstrained, &smoothing
			}
			writer.write(1, 1)
			assertNetworkAttribute(t, attributeTestDecoder(writer, tt.header), "TAGame.CameraSettingsActor_TA:ProfileSettings", want)
		})
	}
}

func TestNetworkRigidBodyVersions(t *testing.T) {
	for _, netVersion := range []int{6, 7} {
		t.Run(fmt.Sprint(netVersion), func(t *testing.T) {
			writer := attributeTestWriter()
			writer.write(0, 1)
			writer.vector(netVersion, 100, -200, 300)
			if netVersion < 7 {
				writer.write(32768, 16)
				writer.write(32767, 16)
				writer.write(65535, 16)
			} else {
				writer.write(3, 2)
				for i := 0; i < 3; i++ {
					writer.write(131072, 18)
				}
			}
			writer.vector(netVersion, 0, 100, -200)
			writer.vector(netVersion, -100, 0, 200)
			writer.write(1, 1)
			decoder := attributeTestDecoder(writer, rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: netVersion})
			value, err := decoder.readAttribute("TAGame.RBActor_TA:ReplicatedRBState")
			if err != nil {
				t.Fatal(err)
			}
			body := value.(rocket.RigidBodyState)
			if body.Sleeping || body.Location != (rocket.Vector3f{X: 1, Y: -2, Z: 3}) ||
				body.LinearVelocity == nil || *body.LinearVelocity != (rocket.Vector3f{Y: 1, Z: -2}) ||
				body.AngularVelocity == nil || *body.AngularVelocity != (rocket.Vector3f{X: -1, Z: 2}) {
				t.Fatalf("unexpected body: %+v", body)
			}
			if netVersion < 7 {
				if body.Rotation.X != 0 || body.Rotation.Y != -1.0/32767 ||
					body.Rotation.Z != 1 || body.Rotation.W != 0 {
					t.Fatalf("unexpected old rotation: %+v", body.Rotation)
				}
			} else if math.Abs(float64(body.Rotation.X)) > 1e-5 ||
				math.Abs(float64(body.Rotation.Y)) > 1e-5 ||
				math.Abs(float64(body.Rotation.Z)) > 1e-5 ||
				math.Abs(float64(body.Rotation.W-1)) > 1e-5 {
				t.Fatalf("unexpected compressed rotation: %+v", body.Rotation)
			}
			if sentinel, err := decoder.reader.readBit(); err != nil || !sentinel {
				t.Fatalf("incorrect rigid body length: %v, %v", sentinel, err)
			}
		})
	}
}

func TestNetworkLoadoutVersions(t *testing.T) {
	for _, version := range []byte{10, 11, 16, 17, 19, 22} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			writer := attributeTestWriter()
			writer.write(uint64(version), 8)
			for i := uint64(1); i <= 7; i++ {
				writer.write(i, 32)
			}
			want := rocket.Loadout{
				Version: version, Body: 1, Decal: 2, Wheels: 3, RocketTrail: 4, Antenna: 5, Topper: 6, Unknown1: 7,
			}
			optional := func(value uint32) *uint32 {
				writer.write(uint64(value), 32)
				return &value
			}
			if version > 10 {
				want.Unknown2 = optional(8)
			}
			if version >= 16 {
				want.EngineAudio, want.Trail, want.GoalExplosion = optional(9), optional(10), optional(11)
			}
			if version >= 17 {
				want.Banner = optional(12)
			}
			if version >= 19 {
				want.ProductID = optional(13)
			}
			if version >= 22 {
				for i := uint64(14); i <= 16; i++ {
					writer.write(i, 32)
				}
				want.Extra = []uint32{14, 15, 16}
			}
			writer.write(1, 1)
			header := rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: 10}
			assertNetworkAttribute(t, attributeTestDecoder(writer, header), "TAGame.PRI_TA:ClientLoadout", want)
		})
	}
}

func TestNetworkOnlineProductVersions(t *testing.T) {
	old := rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 17}
	newer := rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: 10}
	tests := []struct {
		name    string
		header  rocket.ReplayHeader
		product string
		payload func(*networkTestWriter)
		value   interface{}
	}{
		{"old absent color", old, "TAGame.ProductAttribute_UserColor_TA", func(w *networkTestWriter) { w.write(0, 1) }, nil},
		{"old color", old, "TAGame.ProductAttribute_UserColor_TA", func(w *networkTestWriter) {
			w.write(1, 1)
			w.write(123, 31)
		}, uint32(123)},
		{"new color", newer, "TAGame.ProductAttribute_UserColor_TA", func(w *networkTestWriter) {
			w.write(0xff00ff00, 32)
		}, uint32(0xff00ff00)},
		{"old paint", old, "TAGame.ProductAttribute_Painted_TA", func(w *networkTestWriter) { w.bounded(13, 14) }, uint32(13)},
		{"new paint", newer, "TAGame.ProductAttribute_Painted_TA", func(w *networkTestWriter) { w.write(123, 31) }, uint32(123)},
		{"title", newer, "TAGame.ProductAttribute_TitleID_TA", func(w *networkTestWriter) { w.text("winner") }, "winner"},
		{"certified", newer, "TAGame.ProductAttribute_Certified_TA", func(w *networkTestWriter) {
			w.write(8, 31)
			w.write(123, 32)
		}, rocket.ProductCertification{Stat: 8, Value: 123}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := attributeTestWriter()
			writer.write(1, 8)
			writer.write(1, 8)
			writer.write(1, 1)
			writer.write(0, 32)
			tt.payload(writer)
			writer.write(1, 1)
			decoder := attributeTestDecoder(writer, tt.header)
			decoder.replay.Objects = []string{tt.product}
			want := rocket.OnlineLoadout{{{Flag: true, ObjectIndex: 0, Value: tt.value}}}
			assertNetworkAttribute(t, decoder, "TAGame.PRI_TA:ClientLoadoutOnline", want)
		})
	}
}

func TestNetworkTextAndServerID(t *testing.T) {
	header := rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: 10}
	t.Run("UTF-16", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(0xfffffffd, 32)
		writer.write(0x03a9, 16)
		writer.write('!', 16)
		writer.write(0, 16)
		writer.write(1, 1)
		assertNetworkAttribute(t, attributeTestDecoder(writer, header), "Engine.PlayerReplicationInfo:PlayerName", "\u03a9!")
	})
	for _, tt := range []struct {
		name, version string
		isText        bool
	}{
		{"old", "221119.00000.000000", false},
		{"new", "221120.42953.406184", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writer := attributeTestWriter()
			var want interface{}
			if tt.isText {
				writer.text("server")
				want = "server"
			} else {
				writer.write(123, 64)
				want = uint64(123)
			}
			writer.write(1, 1)
			decoder := attributeTestDecoder(writer, header)
			decoder.replay.Properties["BuildVersion"] = &rocket.Property{Value: tt.version}
			assertNetworkAttribute(t, decoder, "ProjectX.GRI_X:GameServerID", want)
		})
	}
}

func TestNetworkAdditionalAttributeTypes(t *testing.T) {
	header := rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: 10}
	t.Run("legacy pickup", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(1, 1)
		writer.write(42, 32)
		writer.write(1, 1)
		writer.write(1, 1)
		instigator := int32(42)
		assertNetworkAttribute(t, attributeTestDecoder(writer, header), "TAGame.VehiclePickup_TA:ReplicatedPickupData",
			rocket.Pickup{Instigator: &instigator, PickedUp: true})
	})
	t.Run("replicated boost", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.bytes([]byte{3, 100, 4, 5})
		writer.write(1, 1)
		assertNetworkAttribute(t, attributeTestDecoder(writer, header), "TAGame.CarComponent_Boost_TA:ReplicatedBoost",
			rocket.ReplicatedBoost{GrantCount: 3, BoostAmount: 100, Unknown1: 4, Unknown2: 5})
	})
	t.Run("private match", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.text("mutators")
		writer.write(2, 32)
		writer.write(8, 32)
		writer.text("match")
		writer.text("password")
		writer.write(1, 1)
		writer.write(1, 1)
		want := rocket.PrivateMatchSettings{
			Mutators: "mutators", JoinableBy: 2, MaxPlayers: 8, GameName: "match", Password: "password", Flag: true,
		}
		assertNetworkAttribute(t, attributeTestDecoder(writer, header), "TAGame.GameEvent_SoccarPrivate_TA:MatchSettings", want)
	})
	for _, tt := range []struct {
		name   string
		header rocket.ReplayHeader
		width  byte
	}{
		{"old game mode", rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 11}, 2},
		{"new game mode", rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 12}, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writer := attributeTestWriter()
			writer.write(3, int(tt.width))
			writer.write(1, 1)
			assertNetworkAttribute(t, attributeTestDecoder(writer, tt.header), "TAGame.GameEvent_TA:GameMode",
				rocket.GameMode{Width: tt.width, Value: 3})
		})
	}
}

func TestNetworkAttributeErrors(t *testing.T) {
	header := rocket.ReplayHeader{EngineVersion: 868, LicenseeVersion: 28, NetVersion: 10}
	t.Run("unknown platform", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(3, 8)
		_, err := attributeTestDecoder(writer, header).readAttribute("Engine.PlayerReplicationInfo:UniqueId")
		if err == nil || !strings.Contains(err.Error(), "unsupported unique ID system 3") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("truncated text", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(0x7fffffff, 32)
		_, err := attributeTestDecoder(writer, header).readAttribute("Engine.PlayerReplicationInfo:PlayerName")
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("got %v, want unexpected EOF", err)
		}
	})
	t.Run("invalid build version", func(t *testing.T) {
		for _, property := range []*rocket.Property{nil, {Value: int32(2)}} {
			decoder := attributeTestDecoder(attributeTestWriter(), header)
			decoder.replay.Properties["BuildVersion"] = property
			_, err := decoder.readAttribute("ProjectX.GRI_X:GameServerID")
			if err == nil || !strings.Contains(err.Error(), "BuildVersion") {
				t.Fatalf("got %v", err)
			}
		}
	})
	t.Run("truncated compound", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(1, 8)
		_, err := attributeTestDecoder(writer, header).readAttribute("TAGame.Car_TA:TeamPaint")
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("got %v, want unexpected EOF", err)
		}
	})
	t.Run("unknown product", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(1, 8)
		writer.write(1, 8)
		writer.write(0, 1)
		writer.write(0, 32)
		decoder := attributeTestDecoder(writer, header)
		decoder.replay.Objects = []string{"TAGame.ProductAttribute_Unknown_TA"}
		_, err := decoder.readAttribute("TAGame.PRI_TA:ClientLoadoutOnline")
		if err == nil || !strings.Contains(err.Error(), "unsupported product attribute") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("invalid quaternion", func(t *testing.T) {
		writer := attributeTestWriter()
		writer.write(1, 1)
		writer.vector(10, 0, 0, 0)
		writer.write(3, 2)
		writer.write(0, 18)
		writer.write(0, 18)
		writer.write(0, 18)
		_, err := attributeTestDecoder(writer, header).readAttribute("TAGame.RBActor_TA:ReplicatedRBState")
		if err == nil || !strings.Contains(err.Error(), "invalid compressed quaternion") {
			t.Fatalf("got %v", err)
		}
	})
}
