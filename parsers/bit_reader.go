package parsers

import (
	"fmt"
	"io"
	"math"
	"math/bits"
)

type bitReader struct {
	data       []byte
	byteOffset int
	bitOffset  uint8
}

func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data}
}

func (reader *bitReader) readBit() (bool, error) {
	value, err := reader.readBits(1)
	return value != 0, err
}

func (reader *bitReader) readBits(width int) (uint64, error) {
	if width < 0 || width > 64 {
		return 0, fmt.Errorf("invalid bit width: %d", width)
	}
	if width == 0 {
		return 0, nil
	}
	if (int(reader.bitOffset)+width+7)/8 > len(reader.data)-reader.byteOffset {
		return 0, io.ErrUnexpectedEOF
	}

	var value uint64
	for consumed := 0; consumed < width; {
		count := 8 - int(reader.bitOffset)
		if count > width-consumed {
			count = width - consumed
		}
		mask := byte((uint16(1) << uint(count)) - 1)
		value |= uint64((reader.data[reader.byteOffset]>>reader.bitOffset)&mask) << uint(consumed)
		consumed += count
		reader.bitOffset += uint8(count)
		if reader.bitOffset == 8 {
			reader.byteOffset++
			reader.bitOffset = 0
		}
	}
	return value, nil
}

func (reader *bitReader) readInt(width int) (int64, error) {
	value, err := reader.readBits(width)
	if err != nil {
		return 0, err
	}
	if width > 0 && width < 64 && value&(uint64(1)<<uint(width-1)) != 0 {
		value |= ^uint64(0) << uint(width)
	}
	return int64(value), nil
}

func (reader *bitReader) readFloat32() (float32, error) {
	value, err := reader.readBits(32)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(uint32(value)), nil
}

// readBoundedUint reads a value in [0, max), using an extra bit only when needed.
func (reader *bitReader) readBoundedUint(max uint64) (uint64, error) {
	if max == 0 {
		return 0, fmt.Errorf("invalid exclusive maximum: %d", max)
	}
	width := bits.Len64(max) - 1
	value, err := reader.readBits(width)
	if err != nil {
		return 0, err
	}
	highBit := uint64(1) << uint(width)
	if highBit < max-value {
		set, err := reader.readBit()
		if err != nil {
			return 0, err
		}
		if set {
			value |= highBit
		}
	}
	return value, nil
}

type vector3i struct {
	X, Y, Z int32
}

func (reader *bitReader) readVector3i(netVersion int) (vector3i, error) {
	limit := uint64(20)
	if netVersion >= 7 {
		limit = 22
	}
	size, err := reader.readBoundedUint(limit)
	if err != nil {
		return vector3i{}, fmt.Errorf("vector size: %w", err)
	}
	bias := int32(1) << uint(size+1)
	var components [3]int32
	for i := range components {
		value, err := reader.readBits(int(size) + 2)
		if err != nil {
			return vector3i{}, fmt.Errorf("vector component %d: %w", i, err)
		}
		components[i] = int32(value) - bias
	}
	return vector3i{X: components[0], Y: components[1], Z: components[2]}, nil
}

type rotation struct {
	Yaw, Pitch, Roll *int8
}

func (reader *bitReader) readRotation() (rotation, error) {
	var axes [3]*int8
	for i, name := range []string{"yaw", "pitch", "roll"} {
		present, err := reader.readBit()
		if err != nil {
			return rotation{}, fmt.Errorf("rotation %s presence: %w", name, err)
		}
		if present {
			value, err := reader.readInt(8)
			if err != nil {
				return rotation{}, fmt.Errorf("rotation %s: %w", name, err)
			}
			axis := int8(value)
			axes[i] = &axis
		}
	}
	return rotation{Yaw: axes[0], Pitch: axes[1], Roll: axes[2]}, nil
}
