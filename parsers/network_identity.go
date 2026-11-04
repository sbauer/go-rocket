package parsers

import (
	"fmt"

	"github.com/sbauer/go-rocket"
)

func (decoder *networkDecoder) uniqueID(bits *attributeBits) rocket.UniqueID {
	return decoder.uniqueIDForSystem(bits, bits.byte())
}

func (decoder *networkDecoder) uniqueIDForSystem(bits *attributeBits, system byte) rocket.UniqueID {
	id := rocket.UniqueID{SystemID: system}
	switch system {
	case 0:
		id.RemoteID = uint32(bits.uint(24))
	case 1, 4, 5:
		id.RemoteID = bits.uint(64)
	case 2:
		name := bits.fixedText(16)
		unknownLength := 8
		if decoder.replay.HeaderInfo.NetVersion >= 1 {
			unknownLength = 16
		}
		id.RemoteID = rocket.PlayStationID{Name: name, Unknown: bits.bytes(unknownLength), OnlineID: bits.uint(64)}
	case 6:
		id.RemoteID = rocket.SwitchID{OnlineID: bits.uint(64), Unknown: bits.bytes(24)}
	case 7:
		psynet := rocket.PsyNetID{OnlineID: bits.uint(64)}
		if decoder.replay.HeaderInfo.NetVersion < 10 {
			psynet.Unknown = bits.bytes(24)
		}
		id.RemoteID = psynet
	case 11:
		id.RemoteID = bits.text()
	default:
		if bits.err == nil {
			bits.err = fmt.Errorf("unsupported unique ID system %d", system)
		}
	}
	id.LocalID = bits.byte()
	return id
}

func (decoder *networkDecoder) reservation(bits *attributeBits) rocket.Reservation {
	reservation := rocket.Reservation{Number: byte(bits.uint(3)), UniqueID: decoder.uniqueID(bits)}
	if reservation.UniqueID.SystemID != 0 {
		name := bits.text()
		reservation.Name = &name
	} else if split, ok := reservation.UniqueID.RemoteID.(uint32); ok && split != 0 {
		var nameBytes []byte
		terminated := false
		for len(nameBytes) < 255 && bits.err == nil {
			character := bits.byte()
			if character == 0 {
				terminated = true
				break
			}
			nameBytes = append(nameBytes, character)
		}
		if !terminated && bits.err == nil {
			bits.err = fmt.Errorf("unterminated reservation name")
		}
		name := decodeWindows1252(nameBytes)
		reservation.Name = &name
	}
	reservation.Flag1, reservation.Flag2 = bits.boolean(), bits.boolean()
	if decoder.versionAtLeast(868, 12, 0) {
		extra := byte(bits.uint(6))
		reservation.Extra = &extra
	}
	return reservation
}
