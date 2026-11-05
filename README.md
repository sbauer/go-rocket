# Go Rocket

Go Rocket is an experimental Go library for reading Rocket League replay files.
It parses replay metadata and body tables, with optional gameplay frame decoding.

## Installation

Requires Go 1.14 or newer.

```sh
go get github.com/sbauer/go-rocket
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/sbauer/go-rocket/parsers"
	"github.com/sbauer/go-rocket/source"
)

func main() {
	replay, err := parsers.Parse(source.FromFile("replay.replay"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Replay type:", replay.HeaderInfo.ClassName)
	if mapName, ok := replay.Properties["MapName"]; ok {
		fmt.Println("Map:", mapName.Value)
	}
	fmt.Println("Keyframes:", len(replay.Keyframes))
}
```

Use `source.FromBytes(data)` instead to parse an in-memory replay.

To decode network frames, use `parsers.ParseWithOptions` with
`parsers.ParseOptions{DecodeNetwork: true}`. This retains `Replay.NetworkData`
and populates `Replay.Frames`; the default `Parse` skips frame decoding.
Attribute names can be resolved through `Replay.Objects[update.ObjectIndex]`,
and `update.Value` contains a Go primitive or a typed value from the `rocket`
package.

## Available data

- Header versions and replay properties, including scores, player stats, and goals.
  Scalar and nested `StructProperty` values are in `Property.Value`; array
  elements are in `Property.Groups`. The Go types for each property type are
  documented on `rocket.Property`.
- Levels, keyframes, debug data, tickmarks, packages, objects, and names.
- Class indices and network caches.
- The raw, undecoded network stream in `Replay.NetworkData`.
- When requested, `Replay.Frames` contains actor spawns, deletions, and attribute updates in stream order.

Replay text is decoded as UTF-16 or Windows-1252; identifiers use UTF-8.
Strings are returned without their serialized null terminators.

`Replay.OrderedProperties` preserves header order and duplicate names. Array
groups have `OrderedProperties`, and struct values have `OrderedFields`.
Their corresponding `Properties` or `Fields` maps provide the last occurrence
of each name. Both views share property pointers, but adding, removing, or
renaming entries does not automatically synchronize them.

Header and body CRCs are checked by `parsers.Parse` by default. Use
`parsers.ParseWithOptions` with `ParseOptions{CRCValidation: parsers.CRCValidationNever}`
to skip checks, or `CRCValidationOnParseError` to check a section only if parsing it fails.

## Limitations

- Network decoding supports common physics, player ID, loadout, and match
  attributes, but some classes, attributes, and version-specific formats are
  still unsupported. Unsupported data returns an error rather than being skipped.
- Some property types and replay version quirks are not supported.

## Tests

Run `go test ./...` to run the test suite, which includes the replays in `samples/`.
Sources and coverage for the additional fixtures are documented in
[`samples/boxcars/README.md`](samples/boxcars/README.md). With Go 1.18 or newer,
the parser fuzz target can be run with `go test ./parsers -fuzz FuzzParseReplay`.
