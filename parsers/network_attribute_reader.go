package parsers

import (
	"fmt"
	"math"

	"github.com/sbauer/go-rocket"
)

// attributeBits retains the first error so compound values cannot hide truncation.
type attributeBits struct {
	reader *bitReader
	err    error
}

func (bits *attributeBits) uint(width int) uint64 {
	if bits.err != nil {
		return 0
	}
	var value uint64
	value, bits.err = bits.reader.readBits(width)
	return value
}

func (bits *attributeBits) boolean() bool {
	return bits.uint(1) != 0
}

func (bits *attributeBits) byte() byte {
	return byte(bits.uint(8))
}

func (bits *attributeBits) int32() int32 {
	return int32(bits.uint(32))
}

func (bits *attributeBits) uint32() uint32 {
	return uint32(bits.uint(32))
}

func (bits *attributeBits) float32() float32 {
	return math.Float32frombits(bits.uint32())
}

func (bits *attributeBits) bytes(count int) []byte {
	if bits.err != nil {
		return nil
	}
	var value []byte
	value, bits.err = bits.reader.readBytes(count)
	return value
}

func (bits *attributeBits) text() string {
	if bits.err != nil {
		return ""
	}
	var value string
	value, bits.err = bits.reader.readText()
	return value
}

func (bits *attributeBits) fixedText(count int) string {
	if bits.err != nil {
		return ""
	}
	var value string
	value, bits.err = bits.reader.readFixedText(count)
	return value
}

func (bits *attributeBits) activeActor() rocket.ActiveActor {
	return rocket.ActiveActor{Active: bits.boolean(), ActorID: bits.int32()}
}

func (bits *attributeBits) optionalInt32() *int32 {
	if !bits.boolean() || bits.err != nil {
		return nil
	}
	value := bits.int32()
	return &value
}

func (bits *attributeBits) vector(netVersion int) rocket.Vector3f {
	if bits.err != nil {
		return rocket.Vector3f{}
	}
	value, err := bits.reader.readVector3i(netVersion)
	bits.err = err
	return rocket.Vector3f{X: float32(value.X) / 100, Y: float32(value.Y) / 100, Z: float32(value.Z) / 100}
}

func (bits *attributeBits) rotation() rocket.Rotation {
	if bits.err != nil {
		return rocket.Rotation{}
	}
	value, err := bits.reader.readRotation()
	bits.err = err
	return rocket.Rotation{Yaw: value.Yaw, Pitch: value.Pitch, Roll: value.Roll}
}

func (bits *attributeBits) quaternion(netVersion int) rocket.Quaternion {
	var values [4]float32
	if netVersion < 7 {
		for i := 0; i < 3; i++ {
			values[i] = float32(int32(bits.uint(16))-32768) / 32767
		}
	} else {
		largest := int(bits.uint(2))
		var sum float64
		for i := 0; i < 4; i++ {
			if i != largest {
				component := (float64(bits.uint(18))/262143*2 - 1) / math.Sqrt2
				values[i] = float32(component)
				sum += component * component
			}
		}
		if sum > 1.00001 && bits.err == nil {
			bits.err = fmt.Errorf("invalid compressed quaternion")
		} else {
			values[largest] = float32(math.Sqrt(math.Max(0, 1-sum)))
		}
	}
	return rocket.Quaternion{X: values[0], Y: values[1], Z: values[2], W: values[3]}
}

func (bits *attributeBits) bounded(max uint64) uint64 {
	if bits.err != nil {
		return 0
	}
	var value uint64
	value, bits.err = bits.reader.readBoundedUint(max)
	return value
}
