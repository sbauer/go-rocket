package parsers

import (
	"fmt"
	"io"
	"strings"

	"github.com/sbauer/go-rocket"
)

func (decoder *networkDecoder) readAttribute(name string) (interface{}, error) {
	switch name {
	case "Engine.Actor:bBlockActors", "Engine.Actor:bCollideActors", "Engine.Actor:bCollideWorld",
		"Engine.Actor:bHidden", "Engine.Actor:bNetOwner", "Engine.Actor:bTearOff",
		"Engine.GameReplicationInfo:bMatchIsOver", "Engine.Pawn:bFastAttachedMove",
		"Engine.Pawn:bIsCrouched", "Engine.Pawn:bUsedByMatinee",
		"Engine.PlayerReplicationInfo:bAdmin", "Engine.PlayerReplicationInfo:bBot",
		"Engine.PlayerReplicationInfo:bIsSpectator", "Engine.PlayerReplicationInfo:bReadyToPlay",
		"Engine.PlayerReplicationInfo:bTimedOut", "Engine.PlayerReplicationInfo:bWaitingPlayer",
		"ProjectX.GRI_X:bGameStarted", "TAGame.Ball_Haunted_TA:bIsBallBeamed",
		"TAGame.Ball_TA:bPossessionEnabled", "TAGame.Ball_TA:bWarnBallReset",
		"TAGame.CameraSettingsActor_TA:bMouseCameraToggleEnabled",
		"TAGame.CameraSettingsActor_TA:bUsingBehindView", "TAGame.CameraSettingsActor_TA:bUsingSecondaryCamera",
		"TAGame.CameraSettingsActor_TA:bUsingSwivel", "TAGame.Car_TA:bUnlimitedJumps",
		"TAGame.Car_TA:bUnlimitedTimeForDodge", "TAGame.CarComponent_Boost_TA:bNoBoost",
		"TAGame.CarComponent_Boost_TA:bRechargeGroundOnly", "TAGame.CarComponent_Boost_TA:bUnlimitedBoost",
		"TAGame.CarComponent_FlipCar_TA:bFlipRight", "TAGame.GameEvent_Soccar_TA:bAllowHonorDuels",
		"TAGame.GameEvent_Soccar_TA:bBallHasBeenHit", "TAGame.GameEvent_Soccar_TA:bClubMatch",
		"TAGame.GameEvent_Soccar_TA:bDisableCrowdSound", "TAGame.GameEvent_Soccar_TA:bFullClubMatch",
		"TAGame.GameEvent_Soccar_TA:bFullMatchWinnerDecided", "TAGame.GameEvent_Soccar_TA:bGoalsEnabled",
		"TAGame.GameEvent_Soccar_TA:bMatchCreatorAdminEnabled", "TAGame.GameEvent_Soccar_TA:bMatchEnded",
		"TAGame.GameEvent_Soccar_TA:bNoContest", "TAGame.GameEvent_Soccar_TA:bOverTime",
		"TAGame.GameEvent_Soccar_TA:bReadyToStartGame", "TAGame.GameEvent_Soccar_TA:bShouldSpawnGoalIndicators",
		"TAGame.GameEvent_Soccar_TA:bShowIntroScene", "TAGame.GameEvent_Soccar_TA:bUnlimitedTime",
		"TAGame.GameEvent_TA:bAllowReadyUp", "TAGame.GameEvent_TA:bAlwaysShowMatchTypeLabel",
		"TAGame.GameEvent_TA:bCanVoteToForfeit", "TAGame.GameEvent_TA:bHasLeaveMatchPenalty",
		"TAGame.GameEvent_TA:bIsBotMatch", "TAGame.GameEvent_Team_TA:bDisableMutingOtherTeam",
		"TAGame.GameEvent_Team_TA:bDisableQuickChat", "TAGame.GameEvent_Team_TA:bForfeit",
		"TAGame.GRI_TA:bAllowTargetFind", "TAGame.PlayerStart_Platform_TA:bActive",
		"TAGame.PRI_KnockOut_TA:bIsActiveMVP", "TAGame.PRI_KnockOut_TA:bIsEliminated",
		"TAGame.PRI_TA:bAnonymizeToOpponents", "TAGame.PRI_TA:bAnonymizeToTeammates",
		"TAGame.PRI_TA:bEligibleForHonorDuels", "TAGame.PRI_TA:bIdleBanned",
		"TAGame.PRI_TA:bIsDistracted", "TAGame.PRI_TA:bIsInSplitScreen", "TAGame.PRI_TA:bMatchMVP",
		"TAGame.PRI_TA:bOnlineLoadoutSet", "TAGame.PRI_TA:bOnlineLoadoutsSet",
		"TAGame.PRI_TA:bReady", "TAGame.PRI_TA:bReceivedAnonymizationSettings",
		"TAGame.PRI_TA:bUsingBehindView", "TAGame.PRI_TA:bUsingItems",
		"TAGame.PRI_TA:bUsingSecondaryCamera", "TAGame.PRI_TA:PlayerHistoryValid",
		"TAGame.RBActor_TA:bFrozen", "TAGame.RBActor_TA:bIgnoreSyncing", "TAGame.RBActor_TA:bReplayActor",
		"TAGame.SpecialPickup_BallVelcro_TA:bBroken", "TAGame.SpecialPickup_BallVelcro_TA:bHit",
		"TAGame.SpecialPickup_Rugby_TA:bBallWelded", "TAGame.Vehicle_TA:bDriving",
		"TAGame.Vehicle_TA:bHasPostMatchCelebration", "TAGame.Vehicle_TA:bPodiumMode",
		"TAGame.Vehicle_TA:bReplicatedHandbrake", "TAGame.VehiclePickup_TA:bNoPickup":
		return decoder.reader.readBit()
	case "Engine.Pawn:RemoteViewPitch", "Engine.PlayerReplicationInfo:Ping",
		"TAGame.Ball_Breakout_TA:LastTeamTouch", "TAGame.Ball_Haunted_TA:DeactivatedGoalIndex",
		"TAGame.Ball_Haunted_TA:LastTeamTouch", "TAGame.Ball_Haunted_TA:ReplicatedBeamBrokenValue",
		"TAGame.Ball_Haunted_TA:TotalActiveBeams", "TAGame.Ball_TA:HitTeamNum",
		"TAGame.BallKeepUpComponent_TA:KeepUpState",
		"TAGame.CameraSettingsActor_TA:CameraPitch", "TAGame.CameraSettingsActor_TA:CameraYaw",
		"TAGame.CameraSettingsActor_TA:FreeLookAnchorPitch", "TAGame.CameraSettingsActor_TA:FreeLookAnchorYaw",
		"TAGame.Cannon_TA:FireCount", "TAGame.Car_KnockOut_TA:ReplicatedStateChanged",
		"TAGame.CarComponent_Boost_TA:BoostRestriction",
		"TAGame.CarComponent_Boost_TA:ReplicatedBoostAmount", "TAGame.CarComponent_TA:ReplicatedActive",
		"TAGame.GameEvent_Soccar_TA:ReplicatedScoredOnTeam", "TAGame.GameEvent_Soccar_TA:ReplicatedServerPerformanceState",
		"TAGame.GameEvent_TA:ReplicatedStateIndex", "TAGame.PRI_TA:CameraPitch", "TAGame.PRI_TA:CameraYaw",
		"TAGame.PRI_TA:PawnType", "TAGame.PRI_TA:PlayerStatus", "TAGame.PRI_TA:ReplicatedWorstNetQualityBeyondLatency",
		"TAGame.RBActor_TA:TeleportCounter", "TAGame.Vehicle_TA:InputRestriction",
		"TAGame.Vehicle_TA:ReplicatedSteer", "TAGame.Vehicle_TA:ReplicatedThrottle",
		"TAGame.ViralItemActor_TA:ClientFXInfectedType", "TAGame.ViralItemActor_TA:InfectedStatus":
		return decoder.reader.readByte()
	case "Engine.Pawn:HealthMax", "Engine.PlayerReplicationInfo:PlayerID", "Engine.PlayerReplicationInfo:Score",
		"Engine.TeamInfo:Score", "ProjectX.GRI_X:ReplicatedGameMutatorIndex", "ProjectX.GRI_X:ReplicatedGamePlaylist",
		"TAGame.Ball_Breakout_TA:DamageIndex", "TAGame.Ball_TA:GameBallIndex", "TAGame.BallKeepUpComponent_TA:Score",
		"TAGame.Car_KnockOut_TA:ReplicatedStateName", "TAGame.Car_TA:DodgesRefreshedCounter",
		"TAGame.Car_TA:MaxNumJumps", "TAGame.CarComponent_AirActivate_TA:AirActivateCount",
		"TAGame.CarComponent_Boost_TA:UnlimitedBoostRefCount", "TAGame.CarComponent_Dodge_KO_TA:DodgeRotationCompressed",
		"TAGame.CarComponent_Torque_TA:ReplicatedTorqueInput", "TAGame.CrowdActor_TA:ReplicatedCountDownNumber",
		"TAGame.CrowdActor_TA:ReplicatedRoundCountDownNumber", "TAGame.GameEvent_Soccar_TA:GameTime",
		"TAGame.GameEvent_Soccar_TA:MaxScore", "TAGame.GameEvent_Soccar_TA:RoundNum",
		"TAGame.GameEvent_Soccar_TA:SecondsRemaining", "TAGame.GameEvent_Soccar_TA:SeriesLength",
		"TAGame.GameEvent_Soccar_TA:TotalGameBalls",
		"TAGame.GameEvent_TA:BotSkill", "TAGame.GameEvent_TA:ReplicatedGameStateTimeRemaining",
		"TAGame.GameEvent_TA:ReplicatedRoundCountDownNumber", "TAGame.GameEvent_TA:ReplicatedStateName",
		"TAGame.GameEvent_Team_TA:MaxTeamSize", "TAGame.PickupTimer_TA:MaxTimeTillItem",
		"TAGame.PickupTimer_TA:TimeTillItem", "TAGame.PRI_KnockOut_TA:Blocks",
		"TAGame.PRI_KnockOut_TA:DamageCaused", "TAGame.PRI_KnockOut_TA:EliminationOrder",
		"TAGame.PRI_KnockOut_TA:Grabs", "TAGame.PRI_KnockOut_TA:Hits", "TAGame.PRI_KnockOut_TA:KnockoutDeaths",
		"TAGame.PRI_KnockOut_TA:Knockouts", "TAGame.PRI_TA:BallDemolitions", "TAGame.PRI_TA:BallDemolitionSaves",
		"TAGame.PRI_TA:BallTouches", "TAGame.PRI_TA:BoostPickups", "TAGame.PRI_TA:BotBannerProductID",
		"TAGame.PRI_TA:BotProductName", "TAGame.PRI_TA:CarDemolitions", "TAGame.PRI_TA:CarTouches",
		"TAGame.PRI_TA:Dodges", "TAGame.PRI_TA:EquippedTitleStatValue", "TAGame.PRI_TA:KeepUpClears",
		"TAGame.PRI_TA:KeepUpDenials", "TAGame.PRI_TA:KeepUpPossessions", "TAGame.PRI_TA:MatchAerialHits",
		"TAGame.PRI_TA:MatchAssists", "TAGame.PRI_TA:MatchBicycleHits", "TAGame.PRI_TA:MatchBreakoutDamage",
		"TAGame.PRI_TA:MatchCenters", "TAGame.PRI_TA:MatchClears", "TAGame.PRI_TA:MatchCrossbarHits",
		"TAGame.PRI_TA:MatchDemolishes", "TAGame.PRI_TA:MatchEpicSaves", "TAGame.PRI_TA:MatchFirstTouches",
		"TAGame.PRI_TA:MatchFlipResets", "TAGame.PRI_TA:MatchGoals", "TAGame.PRI_TA:MatchHighFives",
		"TAGame.PRI_TA:MatchJuggleHits", "TAGame.PRI_TA:MatchLowFives", "TAGame.PRI_TA:MatchPoolShots",
		"TAGame.PRI_TA:MatchSaves", "TAGame.PRI_TA:MatchScore", "TAGame.PRI_TA:MatchShots",
		"TAGame.PRI_TA:MatchTimesDemolished", "TAGame.PRI_TA:MaxTimeTillItem", "TAGame.PRI_TA:PossessionClears",
		"TAGame.PRI_TA:PossessionDenials", "TAGame.PRI_TA:PossessionSteals", "TAGame.PRI_TA:PowerUpsUsed",
		"TAGame.PRI_TA:SelfDemolitions", "TAGame.PRI_TA:SpectatorShortcut", "TAGame.PRI_TA:TimeTillItem",
		"TAGame.PRI_TA:Title", "TAGame.PRI_TA:TotalXP", "TAGame.RumblePickups_TA:ConcurrentItemCount",
		"TAGame.Team_Soccar_TA:GameScore", "TAGame.Team_TA:Difficulty":
		return decoder.reader.readInt32()
	case "TAGame.GameEvent_TA:MatchStartEpoch", "TAGame.MaxTimeWarningData_TA:EndGameEpochTime",
		"TAGame.MaxTimeWarningData_TA:EndGameWarningEpochTime", "TAGame.PRI_TA:ClubID", "TAGame.Team_TA:ClubID":
		return decoder.reader.readInt(64)
	case "Engine.Actor:DrawScale", "TAGame.Ball_Blade_TA:ExtraSpeed", "TAGame.Ball_Fire_TA:TeamNumChangeTimestamp",
		"TAGame.Ball_God_TA:TargetSpeed", "TAGame.Ball_Spawner_TA:SpawnDelaySeconds",
		"TAGame.Ball_TA:AdditionalCarGroundBounceScaleXY", "TAGame.Ball_TA:AdditionalCarGroundBounceScaleZ",
		"TAGame.Ball_TA:BallHitSpinScale", "TAGame.Ball_TA:MagnusMinSpeed", "TAGame.Ball_TA:ReplicatedAddedCarBounceScale",
		"TAGame.Ball_TA:ReplicatedBallGravityScale", "TAGame.Ball_TA:ReplicatedBallMaxLinearSpeedScale",
		"TAGame.Ball_TA:ReplicatedBallScale", "TAGame.Ball_TA:ReplicatedWorldBounceScale", "TAGame.Cannon_TA:Pitch",
		"TAGame.Car_TA:AddedBallForceMultiplier", "TAGame.Car_TA:AddedCarForceMultiplier",
		"TAGame.Car_TA:ReplicatedCarMaxLinearSpeedScale", "TAGame.Car_TA:ReplicatedCarScale",
		"TAGame.CarComponent_Boost_TA:BoostModifier", "TAGame.CarComponent_Boost_TA:RechargeDelay",
		"TAGame.CarComponent_Boost_TA:RechargeRate", "TAGame.CarComponent_FlipCar_TA:FlipCarTime",
		"TAGame.CarComponent_TA:ReplicatedActivityTime", "TAGame.CarComponent_Torque_TA:TorqueScale",
		"TAGame.CrowdActor_TA:ModifiedNoise", "TAGame.PRI_TA:SteeringSensitivity", "TAGame.PRI_TA:TotalGameTimePlayed",
		"TAGame.PRI_TA:TotalIdleTime", "TAGame.SpecialPickup_BallFreeze_TA:RepOrigSpeed",
		"TAGame.SpecialPickup_BallVelcro_TA:AttachTime", "TAGame.SpecialPickup_BallVelcro_TA:BreakTime",
		"TAGame.Stunlock_TA:MashTime", "TAGame.Stunlock_TA:MaxStunTime", "TAGame.Stunlock_TA:StunTimeRemaining",
		"TAGame.VehiclePickup_TA:RespawnDelayScale":
		return decoder.reader.readFloat32()
	case "Engine.Actor:RemoteRole", "Engine.Actor:Role":
		value, err := decoder.reader.readBits(11)
		return uint16(value), err
	case "Engine.GameReplicationInfo:ServerName", "Engine.PlayerReplicationInfo:PlayerName",
		"Engine.PlayerReplicationInfo:RemoteUserData", "ProjectX.GRI_X:MatchGUID",
		"ProjectX.GRI_X:MatchGuid", "ProjectX.GRI_X:ReplicatedServerRegion",
		"TAGame.GameEvent_TA:RichPresenceString", "TAGame.GRI_TA:LanMatchGUID",
		"TAGame.GRI_TA:NewDedicatedServerIP", "TAGame.PRI_TA:AnonymizedName",
		"TAGame.PRI_TA:CurrentVoiceRoom", "TAGame.PRI_TA:EpicPUID", "TAGame.Team_TA:CustomTeamName":
		return decoder.reader.readText()
	case "ProjectX.GRI_X:GameServerID":
		if property, present := decoder.replay.Properties["BuildVersion"]; present {
			if property == nil {
				return nil, fmt.Errorf("BuildVersion property is nil")
			}
			version, ok := property.Value.(string)
			if !ok {
				return nil, fmt.Errorf("BuildVersion property is not a string")
			}
			if version >= "221120.42953.406184" {
				return decoder.reader.readText()
			}
		}
		return decoder.reader.readBits(64)
	case "Engine.GameReplicationInfo:GameClass", "Engine.Pawn:PlayerReplicationInfo", "Engine.PlayerReplicationInfo:Team",
		"Engine.ReplicatedActor_ORS:ReplicatedOwner", "TAGame.Ball_Blade_TA:ClosestTargetPRI",
		"TAGame.Ball_Blade_TA:TargetPRI", "TAGame.Ball_Spawner_TA:SpawnedBall", "TAGame.Ball_TA:GameEvent",
		"TAGame.Ball_TA:ReplicatedPhysMatOverride", "TAGame.BallKeepUpComponent_TA:BallOwner",
		"TAGame.CameraSettingsActor_TA:PRI", "TAGame.Car_KnockOut_TA:UsedAttackComponent",
		"TAGame.Car_TA:AttachedPickup", "TAGame.Car_TA:RumblePickups", "TAGame.CarComponent_TA:Vehicle",
		"TAGame.CrowdActor_TA:GameEvent", "TAGame.CrowdActor_TA:ReplicatedOneShotSound",
		"TAGame.CrowdManager_TA:GameEvent", "TAGame.CrowdManager_TA:ReplicatedGlobalOneShotSound",
		"TAGame.GameEvent_Soccar_TA:GameWinner", "TAGame.GameEvent_Soccar_TA:MatchWinner",
		"TAGame.GameEvent_Soccar_TA:MVP", "TAGame.GameEvent_Soccar_TA:SubRulesArchetype",
		"TAGame.GameEvent_TA:DemoFXOverride", "TAGame.GameEvent_TA:MatchTypeClass",
		"TAGame.KeepUpIndicator_TA:ComponentOwner", "TAGame.PRI_TA:PersistentCamera",
		"TAGame.PRI_TA:ReplicatedGameEvent", "TAGame.PRI_TA:ViralItemActor",
		"TAGame.RumblePickups_TA:AttachedPickup", "TAGame.SpecialPickup_Football_TA:WeldedBall",
		"TAGame.SpecialPickup_Targeted_TA:Targeted", "TAGame.Stunlock_TA:Car", "TAGame.Team_TA:GameEvent":
		return decoder.readActiveActor()
	default:
		return decoder.readStructuredAttribute(name)
	}
}

func (decoder *networkDecoder) readActiveActor() (rocket.ActiveActor, error) {
	bits := &attributeBits{reader: decoder.reader}
	actor := bits.activeActor()
	return actor, bits.err
}

func (reader *bitReader) readByte() (byte, error) {
	value, err := reader.readBits(8)
	return byte(value), err
}

func (reader *bitReader) readInt32() (int32, error) {
	value, err := reader.readInt(32)
	return int32(value), err
}

func (reader *bitReader) readText() (string, error) {
	return readString(bitByteReader{reader})
}

type bitByteReader struct {
	reader *bitReader
}

func (reader bitByteReader) Read(buffer []byte) (int, error) {
	for i := range buffer {
		value, err := reader.reader.readByte()
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				return i, io.EOF
			}
			return i, err
		}
		buffer[i] = value
	}
	return len(buffer), nil
}

func (reader bitByteReader) Len() int {
	return ((len(reader.reader.data)-reader.reader.byteOffset)*8 - int(reader.reader.bitOffset)) / 8
}

func (reader *bitReader) readBytes(count int) ([]byte, error) {
	data := make([]byte, count)
	_, err := io.ReadFull(bitByteReader{reader}, data)
	return data, err
}

func (reader *bitReader) readFixedText(count int) (string, error) {
	data, err := reader.readBytes(count)
	if err != nil {
		return "", err
	}
	if index := strings.IndexByte(string(data), 0); index >= 0 {
		data = data[:index]
	}
	return decodeWindows1252(data), nil
}

func (decoder *networkDecoder) versionAtLeast(engine, licensee, network int) bool {
	header := decoder.replay.HeaderInfo
	if header.EngineVersion != engine {
		return header.EngineVersion > engine
	}
	if header.LicenseeVersion != licensee {
		return header.LicenseeVersion > licensee
	}
	return header.NetVersion >= network
}
