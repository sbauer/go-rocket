# Additional replay fixtures

These files are from `assets/replays/good/` in
[Boxcars](https://github.com/nickbabcock/boxcars), pinned to commit
`6f843778a116224d0d1aeedf158ccf58d265318c`. See `THIRD_PARTY_NOTICES`
for the source license.

Versions are engine/licensee/network; "absent" means the replay predates the
serialized network-version field. Frame and event expectations were checked
against rrrocket 0.11.8.

| File | Version | Frames | Coverage |
| --- | --- | ---: | --- |
| `no-frames.replay` | 0/0/absent | n/a | Legacy header without frame data |
| `07e9.replay` | 868/11/absent | 319 | 2016 Hoops |
| `7083.replay` | 868/20/2 | 8,346 | Hoops, Switch and split-screen IDs |
| `d52eb.replay` | 868/23/8 | 17,815 | Legacy PsyNet IDs |
| `9a2cd.replay` | 868/26/10 | 2,616 | Haunted mode; PlayStation, Xbox and newer PsyNet IDs |
| `0ca5.replay` | 868/27/10 | 472 | QQ IDs |
| `dropshot_rumble.replay` | 868/32/10 | 468 | Dropshot Rumble |
| `heatseeker.replay` | 868/32/10 | 726 | Heatseeker |
| `3cdd.replay` | 868/32/10 | 753 | Snow Day Territory and a Rumble freeze pickup |
| `MaxNumJumps.replay` | 868/34/12 | 6,065 | 2026 format, nested player IDs and Epic IDs |
