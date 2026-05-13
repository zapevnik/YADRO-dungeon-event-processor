# Dungeon Event Processor

A command-line tool that processes a stream of dungeon events and produces a final report for each player.

## Notes

- Players who registered but never entered the dungeon are marked as DISQUALIFIED by default, 
since the technical requirements require entering the dungeon to receive a fail or success state.

- The typo `imposible` (single 's') in event output is intentional and matches the task specification.

- Input file paths are fixed: `task_data/config.json` and `task_data/events`.

- The last floor (as defined by the `Floors` config value) is treated as the boss floor entry point. Regular floors with monsters are floors 1 through `Floors - 1`.

- An impossible move (event 33) is logged but does not affect the player's state or
terminate the challenge — the player continues normally.

- Going back to a previous floor (event 5) resets the monster kill counter for that floor,
requiring the player to kill all monsters again on the next visit.

## Requirements

- Go 1.22 or later

## Project Structure

```
.
├── main.go
├── go.mod
├── task_data/
│   ├── config.json     # dungeon configuration
│   ├── events          # input events file
│   └── TASK.md         # original task description
└── internal/
    ├── models/         # domain types and constants
    ├── parser/         # config loader and event parser
    ├── processor/      # core event processing logic
    └── report/         # final report generation
```

## Running

From the project root:

```bash
go run cmd/main.go
```

## Running Tests

From the project root:

```bash
# All tests
go test ./...

# With verbose output
go test -v ./...

# Processor package only (tests only here)
go test -v ./internal/processor/

# A single test by name
go test -run TestFullRun_Success ./internal/processor/
```

## Config Format (`config.json`)

```json
{
    "Floors": 2,
    "Monsters": 2,
    "OpenAt": "14:05:00",
    "Duration": 2
}
```

| Field | Description |
|---|---|
| `Floors` | Total number of floors; the last floor is the entry point to the boss floor |
| `Monsters` | Number of monsters on each regular floor |
| `OpenAt` | Dungeon opening time (`HH:MM:SS`) |
| `Duration` | How long the dungeon stays open, in hours |

## Events File Format

Each line follows the pattern: `[HH:MM:SS] <playerID> <eventID> [extraParam]`

```
[14:00:00] 1 1
[14:10:00] 1 2
[14:15:00] 1 11 30
```

## Sample Output

```
[14:00:00] Player [1] registered
[14:10:00] Player [1] entered the dungeon
...

Final report:
[SUCCESS] 1 [00:24:00, 00:05:00, 00:11:00] HP:35
[FAIL] 2 [00:19:00, 00:00:00, 00:00:00] HP:0
[DISQUAL] 3 [00:00:00, 00:00:00, 00:00:00] HP:100
```
