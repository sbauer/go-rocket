
# Go Rocket [![Go Report Card](https://goreportcard.com/badge/github.com/sbauer/go-rocket)](https://goreportcard.com/report/github.com/sbauer/go-rocket)
Go library used to parse Rocket League replay files. Early dev, so things will break.

## Features
* Header and property processing (score, players, goals, etc.)
* Levels, keyframes, debug data, tickmarks, and package, object, and name lists
* Class indices and network caches
* Raw network data in `Replay.NetworkData`
* File and byte-slice sources

Replay text is decoded as UTF-16 or Windows-1252; identifiers use UTF-8.
Strings are returned without their serialized null terminators.

## Work in Progress
* Decoding network frames and actor data
* Header and body CRC validation
* Additional property types and replay version quirks
* Better API
* Stability
* Tests
