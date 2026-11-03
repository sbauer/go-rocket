# Go Rocket

Go Rocket is an experimental Go library for reading Rocket League replay files.
It parses replay metadata and body tables, but does not yet decode gameplay frames.

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

## Available data

- Header versions and replay properties, including scores, player stats, and goals.
  Scalar properties are in `Property.Value`; array elements are in `Property.Groups`.
- Levels, keyframes, debug data, tickmarks, packages, objects, and names.
- Class indices and network caches.
- The raw, undecoded network stream in `Replay.NetworkData`.

Replay text is decoded as UTF-16 or Windows-1252; identifiers use UTF-8.
Strings are returned without their serialized null terminators.

## Limitations

- Network frames and actor data are not decoded.
- Header and body CRCs are not validated.
- Some property types and replay version quirks are not supported.

## Tests

Run `go test ./...` to run the test suite, which includes the replays in `samples/`.
