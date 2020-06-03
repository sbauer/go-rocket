package rocket

// Property is a type used to contain replay properties
type Property struct {
	Name   string
	Value  interface{}
	Type   string
	Groups []*PropertyGroup
}

// PropertyGroup is a type used to contain groups of properties
type PropertyGroup struct {
	Properties map[string]*Property
}
