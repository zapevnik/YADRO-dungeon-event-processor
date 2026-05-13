package models

import "time"

// EventID constants for incoming events.
const (
	EventRegister        = 1
	EventEnterDungeon    = 2
	EventKillMonster     = 3
	EventNextFloor       = 4
	EventPrevFloor       = 5
	EventEnterBossFloor  = 6
	EventKillBoss        = 7
	EventLeaveDungeon    = 8
	EventCannotContinue  = 9
	EventRestoreHealth   = 10
	EventReceiveDamage   = 11
)

// EventID constants for outgoing events.
const (
	OutEventDisqualified  = 31
	OutEventDead          = 32
	OutEventImpossible    = 33
)

// PlayerStatus represents the final challenge outcome.
type PlayerStatus string

const (
	StatusSuccess PlayerStatus = "SUCCESS"
	StatusFail    PlayerStatus = "FAIL"
	StatusDisqual PlayerStatus = "DISQUAL"
)

// PlayerState tracks all mutable state for a single player during the challenge.
type PlayerState struct {
	ID int

	Registered bool
	InDungeon  bool
	Dead       bool
	// Finished means the challenge ended (left, died, disqualified, expired).
	Finished   bool

	HP           int // starts at 100 when entering the dungeon
	CurrentFloor int // 1-indexed; 0 = not in dungeon
	OnBossFloor  bool

	// Tracking floor clear times.
	FloorEnterTime time.Time   // when player entered current floor
	FloorTimes     []time.Duration // time to clear each regular floor

	// Boss tracking.
	BossFloorEnterTime time.Time
	BossKillTime       time.Duration

	// Dungeon-wide timing.
	DungeonEnterTime time.Time
	DungeonEndTime   time.Time

	// Per-floor monster kill counts.
	MonstersKilledOnFloor int

	// Final report state.
	FinalStatus PlayerStatus
	FinalHP     int
	FinalTime   time.Duration
}

// RawEvent is a parsed line from the input stream.
type RawEvent struct {
	Time       time.Time
	PlayerID   int
	EventID    int
	ExtraParam string
}

// OutputLine is an output line to be printed.
type OutputLine struct {
	Time    time.Time
	Message string
}


