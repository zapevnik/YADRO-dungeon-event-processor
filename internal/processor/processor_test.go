package processor

import (
	"testing"
	"time"

	"dungeon-event-processor/internal/models"
	"dungeon-event-processor/internal/parser"
)

// makeConfig builds a Config with pre-parsed time fields, bypassing file I/O.
func makeConfig(floors, monsters int, openAt string, durationHours int) *parser.Config {
	t, _ := time.Parse("15:04:05", openAt)
	return &parser.Config{
		Floors:      floors,
		Monsters:    monsters,
		OpenAt:      openAt,
		Duration:    durationHours,
		OpenAtTime:  t,
		CloseAtTime: t.Add(time.Duration(durationHours) * time.Hour),
	}
}

// ev is a shorthand for constructing a RawEvent, keeping test cases concise.
func ev(timeStr string, playerID, eventID int, extra string) models.RawEvent {
	t, _ := time.Parse("15:04:05", timeStr)
	return models.RawEvent{Time: t, PlayerID: playerID, EventID: eventID, ExtraParam: extra}
}

// messages extracts the Message field from each OutputLine for easier assertions.
func messages(lines []models.OutputLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Message
	}
	return out
}

// containsMsg reports whether any OutputLine carries the given message text.
func containsMsg(lines []models.OutputLine, msg string) bool {
	for _, l := range lines {
		if l.Message == msg {
			return true
		}
	}
	return false
}

func TestRegister_Success(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	out := proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
	})

	if !containsMsg(out, "Player [1] registered") {
		t.Errorf("expected registration message, got: %v", messages(out))
	}
	if !proc.Players()[1].Registered {
		t.Error("player should be marked as Registered")
	}
}

// TestRegister_Duplicate verifies that sending EventRegister twice for the same
// player results in disqualification on the second attempt.
func TestRegister_Duplicate(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:01:00", 1, models.EventRegister, ""),
	})

	if proc.Players()[1].FinalStatus != models.StatusDisqual {
		t.Errorf("expected DISQUAL on duplicate register, got: %s", proc.Players()[1].FinalStatus)
	}
}

// TestEnterDungeon_NotRegistered verifies that a player who skips registration
// is disqualified when attempting to enter the dungeon.
func TestEnterDungeon_NotRegistered(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	out := proc.Process([]models.RawEvent{
		ev("10:05:00", 99, models.EventEnterDungeon, ""),
	})

	if !containsMsg(out, "Player [99] is disqualified") {
		t.Errorf("expected disqualified message, got: %v", messages(out))
	}
}

// TestEnterDungeon_BeforeOpen verifies that entering before OpenAt is rejected.
func TestEnterDungeon_BeforeOpen(t *testing.T) {
	cfg := makeConfig(2, 2, "14:00:00", 2)
	proc := New(cfg)

	out := proc.Process([]models.RawEvent{
		ev("13:00:00", 1, models.EventRegister, ""),
		ev("13:59:00", 1, models.EventEnterDungeon, ""),
	})

	if !containsMsg(out, "Player [1] is disqualified") {
		t.Errorf("expected disqualified for entering before open, got: %v", messages(out))
	}
}

// TestDamageAndDeath verifies that cumulative damage exceeding 100 HP kills the
// player, sets FinalStatus to FAIL, and records HP as 0.
func TestDamageAndDeath(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("10:10:00", 1, models.EventReceiveDamage, "60"),
		ev("10:11:00", 1, models.EventReceiveDamage, "50"),
	})

	ps := proc.Players()[1]
	if ps.FinalStatus != models.StatusFail {
		t.Errorf("expected FAIL after death, got: %s", ps.FinalStatus)
	}
	if ps.FinalHP != 0 {
		t.Errorf("expected HP=0 after death, got: %d", ps.FinalHP)
	}
	if !ps.Dead {
		t.Error("player should be marked as Dead")
	}
}

// TestDamageAfterDeath verifies that all events following player death are
// silently ignored and the death message appears exactly once.
func TestDamageAfterDeath(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	out := proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("10:10:00", 1, models.EventReceiveDamage, "100"),
		ev("10:11:00", 1, models.EventReceiveDamage, "50"),
		ev("10:12:00", 1, models.EventKillMonster, ""),
	})

	count := 0
	for _, l := range out {
		if l.Message == "Player [1] is dead" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 death message, got: %d", count)
	}
}

// TestHealthCap verifies that restoring health beyond the maximum clamps HP at 100.
func TestHealthCap(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("10:06:00", 1, models.EventReceiveDamage, "30"),
		ev("10:07:00", 1, models.EventRestoreHealth, "9999"),
	})

	if proc.Players()[1].HP != 100 {
		t.Errorf("HP should be capped at 100, got: %d", proc.Players()[1].HP)
	}
}

// TestPrevFloor_Impossible verifies that attempting to go back from floor 1
// produces an impossible move event and leaves the player on floor 1.
func TestPrevFloor_Impossible(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	out := proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("10:06:00", 1, models.EventPrevFloor, ""),
	})

	if !containsMsg(out, "Player [1] makes imposible move [5]") {
		t.Errorf("expected impossible move message, got: %v", messages(out))
	}
}

// TestNextFloor_WithoutKillingMonsters verifies that a player cannot advance
// to the next floor while monsters remain alive on the current floor.
func TestNextFloor_WithoutKillingMonsters(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	out := proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("10:06:00", 1, models.EventKillMonster, ""),
		ev("10:07:00", 1, models.EventNextFloor, ""),
	})

	if !containsMsg(out, "Player [1] makes imposible move [4]") {
		t.Errorf("expected impossible move, got: %v", messages(out))
	}
}

// TestFullRun_Success is an end-to-end test covering the complete success path:
// register → enter → clear floor → next floor → boss floor → kill boss → leave.
func TestFullRun_Success(t *testing.T) {
	cfg := makeConfig(2, 2, "14:05:00", 2)
	proc := New(cfg)

	proc.Process([]models.RawEvent{
		ev("14:40:00", 1, models.EventRegister, ""),
		ev("14:40:00", 1, models.EventEnterDungeon, ""),
		ev("14:41:00", 1, models.EventKillMonster, ""),
		ev("14:45:00", 1, models.EventKillMonster, ""),
		ev("14:48:00", 1, models.EventNextFloor, ""),
		ev("14:48:00", 1, models.EventEnterBossFloor, ""),
		ev("14:59:00", 1, models.EventKillBoss, ""),
		ev("15:04:00", 1, models.EventLeaveDungeon, ""),
	})

	if proc.Players()[1].FinalStatus != models.StatusSuccess {
		t.Errorf("expected SUCCESS, got: %s", proc.Players()[1].FinalStatus)
	}
}

// TestFullRun_Fail_NoBoss verifies that leaving the dungeon without killing the
// boss results in FAIL regardless of floor progress.
func TestFullRun_Fail_NoBoss(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	proc.Process([]models.RawEvent{
		ev("10:05:00", 1, models.EventRegister, ""),
		ev("10:06:00", 1, models.EventEnterDungeon, ""),
		ev("10:07:00", 1, models.EventLeaveDungeon, ""),
	})

	if proc.Players()[1].FinalStatus != models.StatusFail {
		t.Errorf("expected FAIL when leaving without clearing, got: %s", proc.Players()[1].FinalStatus)
	}
}

// TestCannotContinue_InDungeon verifies that EventCannotContinue while inside
// the dungeon ends the challenge with DISQUAL.
func TestCannotContinue_InDungeon(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("10:10:00", 1, models.EventCannotContinue, "connection lost"),
	})

	if proc.Players()[1].FinalStatus != models.StatusDisqual {
		t.Errorf("expected DISQUAL on cannot continue, got: %s", proc.Players()[1].FinalStatus)
	}
}

// TestDungeonExpiry verifies that a player still inside when the dungeon closes
// is marked Finished with FAIL status.
func TestDungeonExpiry(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 1)
	proc := New(cfg)

	proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("11:00:00", 1, models.EventKillMonster, ""),
	})

	ps := proc.Players()[1]
	if ps.FinalStatus != models.StatusFail {
		t.Errorf("expected FAIL after dungeon expiry, got: %s", ps.FinalStatus)
	}
	if !ps.Finished {
		t.Error("player should be marked Finished after expiry")
	}
}

// TestReceiveDamage_OutputFormat verifies that the damage amount is included
// in the output message using the correct format.
func TestReceiveDamage_OutputFormat(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	out := proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("10:06:00", 1, models.EventReceiveDamage, "42"),
	})

	if !containsMsg(out, "Player [1] recieved [42] of damage") {
		t.Errorf("expected damage message with amount, got: %v", messages(out))
	}
}

// TestKillMonster_OnBossFloor_Impossible verifies that EventKillMonster on the
// boss floor is rejected, since the boss floor contains no monsters.
func TestKillMonster_OnBossFloor_Impossible(t *testing.T) {
	cfg := makeConfig(2, 2, "10:00:00", 2)
	proc := New(cfg)

	out := proc.Process([]models.RawEvent{
		ev("10:00:00", 1, models.EventRegister, ""),
		ev("10:05:00", 1, models.EventEnterDungeon, ""),
		ev("10:06:00", 1, models.EventKillMonster, ""),
		ev("10:07:00", 1, models.EventKillMonster, ""),
		ev("10:08:00", 1, models.EventNextFloor, ""),
		ev("10:08:00", 1, models.EventEnterBossFloor, ""),
		ev("10:09:00", 1, models.EventKillMonster, ""),
	})

	if !containsMsg(out, "Player [1] makes imposible move [3]") {
		t.Errorf("expected impossible move on boss floor, got: %v", messages(out))
	}
}