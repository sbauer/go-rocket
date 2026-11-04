package parsers

import (
	"fmt"

	"github.com/sbauer/go-rocket"
)

func (decoder *networkDecoder) loadout(bits *attributeBits) rocket.Loadout {
	loadout := rocket.Loadout{
		Version: bits.byte(), Body: bits.uint32(), Decal: bits.uint32(), Wheels: bits.uint32(),
		RocketTrail: bits.uint32(), Antenna: bits.uint32(), Topper: bits.uint32(), Unknown1: bits.uint32(),
	}
	readOptional := func() *uint32 {
		value := bits.uint32()
		return &value
	}
	if loadout.Version > 10 {
		loadout.Unknown2 = readOptional()
	}
	if loadout.Version >= 16 {
		loadout.EngineAudio, loadout.Trail, loadout.GoalExplosion = readOptional(), readOptional(), readOptional()
	}
	if loadout.Version >= 17 {
		loadout.Banner = readOptional()
	}
	if loadout.Version >= 19 {
		loadout.ProductID = readOptional()
	}
	if loadout.Version >= 22 {
		loadout.Extra = []uint32{bits.uint32(), bits.uint32(), bits.uint32()}
	}
	return loadout
}

func (decoder *networkDecoder) onlineLoadout(bits *attributeBits) rocket.OnlineLoadout {
	count := int(bits.byte())
	slots := make(rocket.OnlineLoadout, 0, count)
	for slot := 0; slot < count && bits.err == nil; slot++ {
		attributes := int(bits.byte())
		products := make([]rocket.ProductAttribute, 0, attributes)
		for index := 0; index < attributes && bits.err == nil; index++ {
			product := rocket.ProductAttribute{Flag: bits.boolean(), ObjectIndex: bits.int32()}
			product.Value = decoder.productValue(bits, product.ObjectIndex)
			products = append(products, product)
		}
		slots = append(slots, products)
	}
	return slots
}

func (decoder *networkDecoder) productValue(bits *attributeBits, index int32) interface{} {
	if bits.err != nil {
		return nil
	}
	if index < 0 || int(index) >= len(decoder.replay.Objects) {
		bits.err = fmt.Errorf("invalid product attribute object index %d", index)
		return nil
	}
	switch name := decoder.replay.Objects[index]; name {
	case "TAGame.ProductAttribute_UserColor_TA":
		if decoder.versionAtLeast(868, 23, 8) {
			return bits.uint32()
		}
		if bits.boolean() {
			return uint32(bits.uint(31))
		}
		return nil
	case "TAGame.ProductAttribute_Painted_TA", "TAGame.ProductAttribute_TeamEdition_TA":
		if decoder.versionAtLeast(868, 18, 0) {
			return uint32(bits.uint(31))
		}
		return uint32(bits.bounded(14))
	case "TAGame.ProductAttribute_SpecialEdition_TA":
		return uint32(bits.uint(31))
	case "TAGame.ProductAttribute_TitleID_TA":
		return bits.text()
	case "TAGame.ProductAttribute_Certified_TA":
		return rocket.ProductCertification{Stat: uint32(bits.uint(31)), Value: bits.uint32()}
	default:
		bits.err = fmt.Errorf("unsupported product attribute %q (object %d)", name, index)
		return nil
	}
}
