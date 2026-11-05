package rocket

// Property contains a replay property. Value is int32 for IntProperty, float32
// for FloatProperty, bool for BoolProperty, int64 for QWordProperty, and string
// for StrProperty and NameProperty. ByteProperty values are either a byte, an
// enum string in "kind|value" form, or a single string in version 0/0 replays.
// StructProperty values are StructProperty; ArrayProperty values use Groups.
type Property struct {
	Name   string
	Value  interface{}
	Type   string
	Groups []*PropertyGroup
}

// PropertyGroup is a type used to contain groups of properties
type PropertyGroup struct {
	Properties        map[string]*Property // Last occurrence of each property name.
	OrderedProperties []*Property          // All properties in replay order, including duplicates.
}

// StructProperty contains a named, nested set of replay properties.
type StructProperty struct {
	Name          string
	Fields        map[string]*Property // Last occurrence of each field name.
	OrderedFields []*Property          // All fields in replay order, including duplicates.
}
