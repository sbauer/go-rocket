package parsers

import (
	"errors"
	"io"
	"testing"

	"github.com/sbauer/go-rocket/source"
)

func TestBitReaderUnalignedValues(t *testing.T) {
	reader := newBitReader([]byte{0x7d, 0x6f, 0x5e, 0x4d, 0x3c, 0x2b, 0x1a, 0x09, 0x08})
	if value, err := reader.readBits(3); err != nil || value != 5 {
		t.Fatalf("prefix: got %x, %v", value, err)
	}
	if value, err := reader.readBits(64); err != nil || value != 0x0123456789abcdef {
		t.Fatalf("64-bit value: got %x, %v", value, err)
	}
	if value, err := reader.readBit(); err != nil || !value {
		t.Fatalf("following bit: got %v, %v", value, err)
	}

	reader = newBitReader([]byte{0x05, 0x00, 0x80, 0x02, 0x0e})
	if _, err := reader.readBits(3); err != nil {
		t.Fatal(err)
	}
	if value, err := reader.readFloat32(); err != nil || value != -3.25 {
		t.Fatalf("float: got %v, %v", value, err)
	}
	if value, err := reader.readBit(); err != nil || !value {
		t.Fatalf("following bit: got %v, %v", value, err)
	}
}

func TestBitReaderSignedValues(t *testing.T) {
	reader := newBitReader([]byte{0x1f, 0x03})
	if value, err := reader.readInt(5); err != nil || value != -1 {
		t.Fatalf("5-bit integer: got %d, %v", value, err)
	}
	if value, err := reader.readInt(4); err != nil || value != -8 {
		t.Fatalf("4-bit integer: got %d, %v", value, err)
	}
	if value, err := reader.readBit(); err != nil || !value {
		t.Fatalf("following bit: got %v, %v", value, err)
	}
	reader = newBitReader([]byte{0, 0, 0, 0, 0, 0, 0, 0x80})
	if value, err := reader.readInt(64); err != nil || value != -1<<63 {
		t.Fatalf("64-bit integer: got %d, %v", value, err)
	}
}

func TestBitReaderBoundedIntegers(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		max  uint64
		want uint64
	}{
		{name: "four bits", data: []byte{0x16}, max: 22, want: 6},
		{name: "five bits", data: []byte{0x35}, max: 22, want: 21},
		{name: "power of two", data: []byte{0x0f}, max: 8, want: 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newBitReader(tt.data)
			if value, err := reader.readBoundedUint(tt.max); err != nil || value != tt.want {
				t.Fatalf("got %d, %v; want %d", value, err, tt.want)
			}
			if bit, err := reader.readBit(); err != nil || !bit {
				t.Fatalf("following bit: got %v, %v", bit, err)
			}
		})
	}

	reader := newBitReader([]byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	if value, err := reader.readBoundedUint(^uint64(0)); err != nil || value != ^uint64(0)-1 {
		t.Fatalf("maximum-sized bound: got %x, %v", value, err)
	}
	reader = newBitReader(nil)
	if value, err := reader.readBoundedUint(1); err != nil || value != 0 {
		t.Fatalf("unit bound: got %d, %v", value, err)
	}
	if _, err := reader.readBoundedUint(0); err == nil {
		t.Fatal("expected an error for a zero bound")
	}
}

func TestBitReaderVector3i(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		netVersion int
		want       vector3i
		marker     bool
	}{
		{"small", []byte{0x80, 0x0e}, 10, vector3i{-2, -1, 1}, true},
		{"old prefix", []byte{0x04, 0x80, 0x7f}, 5, vector3i{-32, 0, 31}, true},
		{"new prefix", []byte{0x04, 0x00, 0xff}, 10, vector3i{-32, 0, 31}, true},
		{"old maximum", []byte{0x13, 0, 0, 0, 0, 0xc0, 0xff, 0xff, 0x0f}, 5, vector3i{-1048576, 0, 1048575}, false},
		{"new maximum", []byte{0x15, 0, 0, 0, 0, 0, 0xfc, 0xff, 0xff, 0x03}, 10, vector3i{-4194304, 0, 4194303}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newBitReader(tt.data)
			if value, err := reader.readVector3i(tt.netVersion); err != nil || value != tt.want {
				t.Fatalf("got %+v, %v; want %+v", value, err, tt.want)
			}
			if tt.marker {
				if bit, err := reader.readBit(); err != nil || !bit {
					t.Fatalf("following bit: got %v, %v", bit, err)
				}
			}
		})
	}
}

func TestBitReaderRotation(t *testing.T) {
	reader := newBitReader([]byte{0x01, 0xfd, 0x0b})
	value, err := reader.readRotation()
	if err != nil {
		t.Fatal(err)
	}
	if value.Yaw == nil || *value.Yaw != -128 || value.Pitch != nil || value.Roll == nil || *value.Roll != 127 {
		t.Fatalf("unexpected rotation: %+v", value)
	}
	if bit, err := reader.readBit(); err != nil || !bit {
		t.Fatalf("following bit: got %v, %v", bit, err)
	}
}

func TestBitReaderErrors(t *testing.T) {
	reader := newBitReader([]byte{0xab})
	for _, width := range []int{-1, 65} {
		if _, err := reader.readBits(width); err == nil {
			t.Fatalf("expected an error for width %d", width)
		}
	}
	if _, err := reader.readBits(9); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want unexpected EOF", err)
	}
	if value, err := reader.readBits(8); err != nil || value != 0xab {
		t.Fatalf("failed read advanced the cursor: %x, %v", value, err)
	}
	if _, err := reader.readBit(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want unexpected EOF", err)
	}
	if _, err := newBitReader([]byte{0, 0, 0}).readFloat32(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("float: got %v, want unexpected EOF", err)
	}

	reader = newBitReader([]byte{0})
	if _, err := reader.readBits(4); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.readBoundedUint(22); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("bounded integer: got %v, want unexpected EOF", err)
	}
	if _, err := newBitReader([]byte{0}).readVector3i(10); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("vector: got %v, want unexpected EOF", err)
	}
	reader = newBitReader([]byte{0x80})
	if _, err := reader.readBits(7); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.readRotation(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("rotation: got %v, want unexpected EOF", err)
	}
}

func TestBitReaderReplayFrameTime(t *testing.T) {
	replay, err := Parse(source.FromFile("../samples/first.replay"))
	if err != nil {
		t.Fatal(err)
	}
	reader := newBitReader(replay.NetworkData)
	if value, err := reader.readFloat32(); err != nil || value != 29.55841827392578 {
		t.Fatalf("frame time: got %v, %v", value, err)
	}
	if value, err := reader.readFloat32(); err != nil || value != 0 {
		t.Fatalf("frame delta: got %v, %v", value, err)
	}
}
