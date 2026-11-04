package parsers

import (
	"fmt"

	"github.com/sbauer/go-rocket"
)

func (decoder *networkDecoder) readStructuredAttribute(name string) (interface{}, error) {
	bits := &attributeBits{reader: decoder.reader}
	netVersion := decoder.replay.HeaderInfo.NetVersion
	var value interface{}

	switch name {
	case "TAGame.RBActor_TA:ReplicatedRBState":
		value = decoder.rigidBody(bits)
	case "TAGame.Ball_TA:AirResistance", "TAGame.CarComponent_Dodge_TA:DodgeImpulse",
		"TAGame.CarComponent_Dodge_TA:DodgeTorque", "TAGame.CarComponent_DoubleJump_TA:DoubleJumpImpulse":
		value = bits.vector(netVersion)
	case "Engine.Actor:Rotation":
		value = bits.rotation()
	case "TAGame.CameraSettingsActor_TA:ProfileSettings", "TAGame.PRI_TA:CameraSettings":
		value = decoder.cameraSettings(bits)
	case "TAGame.Car_TA:ReplicatedDemolish":
		value = decoder.demolition(bits)
	case "TAGame.Car_TA:ReplicatedDemolishExtended":
		value = decoder.demolitionExtended(bits)
	case "TAGame.Car_TA:ReplicatedDemolish_CustomFX", "TAGame.Car_TA:ReplicatedDemolishGoalExplosion":
		value = rocket.DemolitionFX{CustomFX: bits.activeActor(), Demolition: decoder.demolition(bits)}
	case "TAGame.Ball_TA:ReplicatedExplosionData":
		value = decoder.explosion(bits)
	case "TAGame.Ball_TA:ReplicatedExplosionDataExtended":
		value = rocket.ExtendedExplosion{Explosion: decoder.explosion(bits), SecondaryActor: bits.activeActor()}
	case "TAGame.VehiclePickup_TA:ReplicatedPickupData":
		value = rocket.Pickup{Instigator: bits.optionalInt32(), PickedUp: bits.boolean()}
	case "TAGame.VehiclePickup_TA:NewReplicatedPickupData":
		value = rocket.PickupNew{Instigator: bits.optionalInt32(), PickedUp: bits.byte()}
	case "TAGame.Ball_TA:ReplicatedExplosionDataReactive", "TAGame.PRI_TA:SkillTier":
		value = rocket.FlaggedByte{Flag: bits.boolean(), Value: bits.byte()}
	case "Engine.PlayerReplicationInfo:UniqueId":
		value = decoder.uniqueID(bits)
	case "TAGame.PRI_TA:PartyLeader":
		system := bits.byte()
		var leader *rocket.UniqueID
		if system != 0 && bits.err == nil {
			id := decoder.uniqueIDForSystem(bits, system)
			leader = &id
		}
		value = leader
	case "ProjectX.GRI_X:Reservations":
		value = decoder.reservation(bits)
	case "TAGame.GameEvent_Soccar_TA:AcceptedHonorDuel":
		value = rocket.HonorDuelChallenge{Challenger: decoder.uniqueID(bits), Defender: decoder.uniqueID(bits)}
	case "TAGame.PRI_TA:ClientLoadout":
		value = decoder.loadout(bits)
	case "TAGame.PRI_TA:ClientLoadouts":
		value = rocket.TeamLoadout{Blue: decoder.loadout(bits), Orange: decoder.loadout(bits)}
	case "TAGame.PRI_TA:ClientLoadoutOnline":
		value = decoder.onlineLoadout(bits)
	case "TAGame.PRI_TA:ClientLoadoutsOnline":
		value = rocket.TeamOnlineLoadouts{
			Blue: decoder.onlineLoadout(bits), Orange: decoder.onlineLoadout(bits),
			Flag1: bits.boolean(), Flag2: bits.boolean(),
		}
	case "TAGame.Car_TA:TeamPaint":
		value = rocket.TeamPaint{
			Team: bits.byte(), PrimaryColor: bits.byte(), AccentColor: bits.byte(),
			PrimaryFinish: bits.uint32(), AccentFinish: bits.uint32(),
		}
	case "TAGame.GameEvent_Soccar_TA:ReplicatedStatEvent":
		value = rocket.StatEvent{Flag: bits.boolean(), ObjectIndex: bits.int32()}
	case "TAGame.Car_TA:ClubColors", "TAGame.Team_TA:ClubColors":
		value = rocket.ClubColors{
			BlueFlag: bits.boolean(), BlueColor: bits.byte(),
			OrangeFlag: bits.boolean(), OrangeColor: bits.byte(),
		}
	case "TAGame.GameEvent_SoccarPrivate_TA:MatchSettings":
		value = rocket.PrivateMatchSettings{
			Mutators: bits.text(), JoinableBy: bits.uint32(), MaxPlayers: bits.uint32(),
			GameName: bits.text(), Password: bits.text(), Flag: bits.boolean(),
		}
	case "TAGame.GameEvent_Soccar_TA:ReplicatedMusicStinger":
		value = rocket.MusicStinger{Flag: bits.boolean(), Cue: bits.uint32(), Trigger: bits.byte()}
	case "TAGame.CarComponent_Boost_TA:ReplicatedBoost":
		value = rocket.ReplicatedBoost{
			GrantCount: bits.byte(), BoostAmount: bits.byte(), Unknown1: bits.byte(), Unknown2: bits.byte(),
		}
	case "TAGame.RumblePickups_TA:PickupInfo":
		value = rocket.PickupInfo{
			Available: [3]rocket.ActiveActor{bits.activeActor(), bits.activeActor(), bits.activeActor()},
			Preview:   bits.boolean(),
		}
	case "TAGame.GameEvent_TA:GameMode":
		width := byte(8)
		if !decoder.versionAtLeast(868, 12, 0) {
			width = 2
		}
		value = rocket.GameMode{Width: width, Value: byte(bits.uint(int(width)))}
	case "TAGame.PRI_TA:PlayerHistoryKey":
		value = uint16(bits.uint(14))
	case "TAGame.PRI_TA:PrimaryTitle", "TAGame.PRI_TA:SecondaryTitle":
		title := rocket.ReplicatedTitle{Flag1: bits.boolean(), Flag2: bits.boolean()}
		for i := range title.Values {
			title.Values[i] = bits.uint32()
		}
		title.Flag3 = bits.boolean()
		value = title
	case "TAGame.PRI_TA:RepStatTitles":
		value = rocket.ReplicatedStatTitle{
			Flag1: bits.boolean(), Name: bits.text(), Flag2: bits.boolean(),
			Index: bits.uint32(), Value: bits.uint32(),
		}
	case "TAGame.RBActor_TA:WeldedInfo":
		value = rocket.WeldedInfo{
			Actor: bits.activeActor(), Offset: bits.vector(netVersion),
			Mass: bits.float32(), Rotation: bits.rotation(),
		}
	case "TAGame.Ball_Breakout_TA:AppliedDamage":
		value = rocket.AppliedDamage{
			ID: bits.byte(), Position: bits.vector(netVersion),
			DamageIndex: bits.int32(), TotalDamage: bits.int32(),
		}
	case "TAGame.BreakOutActor_Platform_TA:DamageState":
		value = rocket.DamageState{
			TileState: bits.byte(), Damaged: bits.boolean(), Offender: bits.int32(),
			BallPosition: bits.vector(netVersion), DirectHit: bits.boolean(), Flag: bits.boolean(),
		}
	case "TAGame.Car_KnockOut_TA:ReplicatedImpulse":
		value = rocket.Impulse{CompressedRotation: bits.int32(), Speed: bits.float32()}
	case "TAGame.Team_TA:LogoData":
		value = rocket.LogoData{LogoID: bits.uint32(), SwapColors: bits.boolean()}
	default:
		return nil, fmt.Errorf("unsupported attribute %q", name)
	}
	if bits.err != nil {
		return nil, bits.err
	}
	return value, nil
}

func (decoder *networkDecoder) rigidBody(bits *attributeBits) rocket.RigidBodyState {
	netVersion := decoder.replay.HeaderInfo.NetVersion
	body := rocket.RigidBodyState{
		Sleeping: bits.boolean(),
		Location: bits.vector(netVersion),
		Rotation: bits.quaternion(netVersion),
	}
	if !body.Sleeping && bits.err == nil {
		linear, angular := bits.vector(netVersion), bits.vector(netVersion)
		body.LinearVelocity, body.AngularVelocity = &linear, &angular
	}
	return body
}

func (decoder *networkDecoder) cameraSettings(bits *attributeBits) rocket.CameraSettings {
	settings := rocket.CameraSettings{
		FOV: bits.float32(), Height: bits.float32(), Angle: bits.float32(),
		Distance: bits.float32(), Stiffness: bits.float32(), Swivel: bits.float32(),
	}
	if decoder.versionAtLeast(868, 20, 0) {
		transition := bits.float32()
		settings.Transition = &transition
	}
	if decoder.versionAtLeast(868, 34, 12) {
		acceleration, deceleration, speed := bits.float32(), bits.float32(), bits.float32()
		unconstrained, smoothing := bits.boolean(), bits.boolean()
		settings.Acceleration, settings.Deceleration, settings.FreeLookSpeed = &acceleration, &deceleration, &speed
		settings.UnconstrainedRotation, settings.FreeLookSmoothing = &unconstrained, &smoothing
	}
	return settings
}

func (decoder *networkDecoder) demolition(bits *attributeBits) rocket.Demolition {
	netVersion := decoder.replay.HeaderInfo.NetVersion
	return rocket.Demolition{
		Attacker: bits.activeActor(), Victim: bits.activeActor(),
		AttackerVelocity: bits.vector(netVersion), VictimVelocity: bits.vector(netVersion),
	}
}

func (decoder *networkDecoder) demolitionExtended(bits *attributeBits) rocket.DemolitionExtended {
	netVersion := decoder.replay.HeaderInfo.NetVersion
	return rocket.DemolitionExtended{
		AttackerPRI: bits.activeActor(), SelfDemo: bits.activeActor(), SelfDemolish: bits.boolean(),
		GoalExplosionOwner: bits.activeActor(), Attacker: bits.activeActor(), Victim: bits.activeActor(),
		AttackerVelocity: bits.vector(netVersion), VictimVelocity: bits.vector(netVersion),
	}
}

func (decoder *networkDecoder) explosion(bits *attributeBits) rocket.Explosion {
	return rocket.Explosion{Source: bits.activeActor(), Location: bits.vector(decoder.replay.HeaderInfo.NetVersion)}
}
