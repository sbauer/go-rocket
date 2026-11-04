package parsers

import (
	"fmt"

	"github.com/sbauer/go-rocket"
)

func decodeAttribute(reader *bitReader, name string) (interface{}, error) {
	switch name {
	case "Engine.Actor:bHidden", "Engine.Actor:bNetOwner", "Engine.Actor:bTearOff":
		return reader.readBit()
	case "Engine.PlayerReplicationInfo:Ping", "TAGame.Vehicle_TA:ReplicatedSteer",
		"TAGame.Vehicle_TA:ReplicatedThrottle":
		value, err := reader.readBits(8)
		return byte(value), err
	case "Engine.PlayerReplicationInfo:PlayerID", "Engine.PlayerReplicationInfo:Score",
		"Engine.TeamInfo:Score":
		value, err := reader.readInt(32)
		return int32(value), err
	case "Engine.Actor:DrawScale":
		return reader.readFloat32()
	case "Engine.Pawn:PlayerReplicationInfo", "Engine.PlayerReplicationInfo:Team":
		active, err := reader.readBit()
		if err != nil {
			return nil, err
		}
		id, err := reader.readInt(32)
		if err != nil {
			return nil, err
		}
		return rocket.ActiveActor{Active: active, ActorID: int32(id)}, nil
	default:
		return nil, fmt.Errorf("unsupported attribute %q", name)
	}
}
