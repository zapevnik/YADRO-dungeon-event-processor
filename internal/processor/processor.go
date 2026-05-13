package processor

import (
	"fmt"
	"strconv"
	"time"

	"dungeon-event-processor/internal/models"
	"dungeon-event-processor/internal/parser"
)

// Processor handles the dungeon event stream and tracks player states.
type Processor struct {
	cfg     *parser.Config
	players map[int]*models.PlayerState
	output  []models.OutputLine

	// Mocked
	notifier Notifier
}



// Notifier sends notifications to players.
type Notifier interface {
	SendBossFloorEntered(playerID int) error
}

// MockNotifier is used because the task mentions notifications to players,
// but does not require a real notification delivery implementation.
type MockNotifier struct{}

func (m MockNotifier) SendBossFloorEntered(playerID int) error {
	return nil
}



// New creates a Processor for the given config.
func New(cfg *parser.Config) *Processor {
	return &Processor{
		cfg:     cfg,
		players: make(map[int]*models.PlayerState),
		notifier: MockNotifier{},
	}
}

// Process handles all events in order and returns output lines.
func (p *Processor) Process(events []models.RawEvent) []models.OutputLine {
	for _, ev := range events {
		p.checkTime(ev.Time)
		p.handleEvent(ev)
	}
	return p.output
}

// Players returns all known player states for report generation.
func (p *Processor) Players() map[int]*models.PlayerState {
	return p.players
}

// checkTime marks all active dungeon players as finished if the dungeon has closed.
func (p *Processor) checkTime(now time.Time) {
	if now.Before(p.cfg.CloseAtTime) {
		return
	}
	for _, ps := range p.players {
		if ps.InDungeon && !ps.Finished {
			ps.Finished = true
			ps.DungeonEndTime = p.cfg.CloseAtTime
			ps.FinalHP = ps.HP
			ps.FinalStatus = models.StatusFail
			ps.FinalTime = ps.DungeonEndTime.Sub(ps.DungeonEnterTime)
		}
	}
}

func (p *Processor) handleEvent(ev models.RawEvent) {
	// Silently ignore all events from finished players (disqualified, dead, left).
	// Register is special: an unregistered player's first event may be register.
	if ev.EventID != models.EventRegister {
		if ps, ok := p.players[ev.PlayerID]; ok && ps.Finished {
			return
		}
	}
	switch ev.EventID {
	case models.EventRegister:
		p.handleRegister(ev)
	case models.EventEnterDungeon:
		p.handleEnterDungeon(ev)
	case models.EventKillMonster:
		p.handleKillMonster(ev)
	case models.EventNextFloor:
		p.handleNextFloor(ev)
	case models.EventPrevFloor:
		p.handlePrevFloor(ev)
	case models.EventEnterBossFloor:
		// Mock
		p.notifier.SendBossFloorEntered(ev.PlayerID)

		p.handleEnterBossFloor(ev)
	case models.EventKillBoss:
		p.handleKillBoss(ev)
	case models.EventLeaveDungeon:
		p.handleLeaveDungeon(ev)
	case models.EventCannotContinue:
		p.handleCannotContinue(ev)
	case models.EventRestoreHealth:
		p.handleRestoreHealth(ev)
	case models.EventReceiveDamage:
		p.handleReceiveDamage(ev)
	}
}

func (p *Processor) getOrCreate(id int) *models.PlayerState {
	if ps, ok := p.players[id]; ok {
		return ps
	}
	ps := &models.PlayerState{
		ID:          id,
		FinalStatus: models.StatusDisqual,
		HP:          100,
		FinalHP:     100,
	}
	p.players[id] = ps
	return ps
}

func (p *Processor) emit(t time.Time, msg string) {
	p.output = append(p.output, models.OutputLine{Time: t, Message: msg})
}

func (p *Processor) emitImpossible(ev models.RawEvent) {
	p.emit(ev.Time, fmt.Sprintf("Player [%d] makes imposible move [%d]", ev.PlayerID, ev.EventID))  // intentional: matches spec
}

// isActive returns true if the player can perform dungeon actions.
func isActive(ps *models.PlayerState) bool {
	return ps.InDungeon && !ps.Finished && !ps.Dead
}

func (p *Processor) handleRegister(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if ps.Registered || ps.InDungeon {
		// Already registered or inside – disqualify.
		p.emit(ev.Time, fmt.Sprintf("Player [%d] is disqualified", ev.PlayerID))
		ps.Finished = true
		ps.FinalStatus = models.StatusDisqual
		return
	}
	ps.Registered = true
	p.emit(ev.Time, fmt.Sprintf("Player [%d] registered", ev.PlayerID))
}

func (p *Processor) handleEnterDungeon(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)

	// Must be registered, not yet in dungeon, not finished, and dungeon must be open.
	if !ps.Registered || ps.InDungeon || ps.Finished {
		p.emit(ev.Time, fmt.Sprintf("Player [%d] is disqualified", ev.PlayerID))
		ps.Finished = true
		ps.FinalStatus = models.StatusDisqual
		return
	}
	if ev.Time.Before(p.cfg.OpenAtTime) || !ev.Time.Before(p.cfg.CloseAtTime) {
		p.emit(ev.Time, fmt.Sprintf("Player [%d] is disqualified", ev.PlayerID))
		ps.Finished = true
		ps.FinalStatus = models.StatusDisqual
		return
	}

	ps.InDungeon = true
	ps.HP = 100
	ps.CurrentFloor = 1
	ps.DungeonEnterTime = ev.Time
	ps.FloorEnterTime = ev.Time
	ps.MonstersKilledOnFloor = 0
	p.emit(ev.Time, fmt.Sprintf("Player [%d] entered the dungeon", ev.PlayerID))
}

func (p *Processor) handleKillMonster(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) || ps.OnBossFloor {
		p.emitImpossible(ev)
		return
	}

	ps.MonstersKilledOnFloor++
	p.emit(ev.Time, fmt.Sprintf("Player [%d] killed the monster", ev.PlayerID))

	// Check if the floor is cleared.
	if ps.MonstersKilledOnFloor >= p.cfg.Monsters {
		elapsed := ev.Time.Sub(ps.FloorEnterTime)
		ps.FloorTimes = append(ps.FloorTimes, elapsed)
		// Floor is now cleared; time is no longer counted from this moment.
		// We mark FloorEnterTime = zero so accidental double-clear can't happen.
		ps.FloorEnterTime = time.Time{}
	}
}

func (p *Processor) handleNextFloor(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) || ps.OnBossFloor {
		p.emitImpossible(ev)
		return
	}
	// Cannot go to next floor if there are still monsters on this floor.
	if ps.MonstersKilledOnFloor < p.cfg.Monsters {
		p.emitImpossible(ev)
		return
	}

	ps.CurrentFloor++
	ps.MonstersKilledOnFloor = 0
	ps.FloorEnterTime = ev.Time
	p.emit(ev.Time, fmt.Sprintf("Player [%d] went to the next floor", ev.PlayerID))
}

func (p *Processor) handlePrevFloor(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) || ps.OnBossFloor || ps.CurrentFloor <= 1 {
		p.emitImpossible(ev)
		return
	}

	ps.CurrentFloor--
	// Going back resets floor progress – monsters there may or may not have
	// been killed before. We track kill count per-floor only for the current
	// visit, so reset it.
	ps.MonstersKilledOnFloor = 0
	ps.FloorEnterTime = ev.Time
	p.emit(ev.Time, fmt.Sprintf("Player [%d] went to the previous floor", ev.PlayerID))
}

func (p *Processor) handleEnterBossFloor(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) {
		p.emitImpossible(ev)
		return
	}
	// Must be on the last regular floor and that floor must be cleared.
	if ps.CurrentFloor != p.cfg.Floors {
		p.emitImpossible(ev)
		return
	}

	ps.OnBossFloor = true
	ps.BossFloorEnterTime = ev.Time
	p.emit(ev.Time, fmt.Sprintf("Player [%d] entered the boss's floor", ev.PlayerID))
}

func (p *Processor) handleKillBoss(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) || !ps.OnBossFloor {
		p.emitImpossible(ev)
		return
	}

	ps.BossKillTime = ev.Time.Sub(ps.BossFloorEnterTime)
	p.emit(ev.Time, fmt.Sprintf("Player [%d] killed the boss", ev.PlayerID))
}

func (p *Processor) handleLeaveDungeon(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) {
		p.emitImpossible(ev)
		return
	}

	// Determine final status.
	allFloorsCleared := len(ps.FloorTimes) >= p.cfg.Floors-1
	bossKilled := ps.BossKillTime > 0

	if allFloorsCleared && bossKilled {
		ps.FinalStatus = models.StatusSuccess
	} else {
		ps.FinalStatus = models.StatusFail
	}

	ps.Finished = true
	ps.DungeonEndTime = ev.Time
	ps.FinalHP = ps.HP
	ps.FinalTime = ev.Time.Sub(ps.DungeonEnterTime)
	p.emit(ev.Time, fmt.Sprintf("Player [%d] left the dungeon", ev.PlayerID))
}

func (p *Processor) handleCannotContinue(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) {
		// Not in dungeon – disqualify.
		p.emit(ev.Time, fmt.Sprintf("Player [%d] is disqualified", ev.PlayerID))
		ps.Finished = true
		ps.FinalStatus = models.StatusDisqual
		return
	}

	ps.Finished = true
	ps.DungeonEndTime = ev.Time
	ps.FinalHP = ps.HP
	ps.FinalStatus = models.StatusDisqual
	ps.FinalTime = ev.Time.Sub(ps.DungeonEnterTime)
	p.emit(ev.Time, fmt.Sprintf("Player [%d] cannot continue due to [%s]", ev.PlayerID, ev.ExtraParam))
}

func (p *Processor) handleRestoreHealth(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) {
		p.emitImpossible(ev)
		return
	}

	amount, err := strconv.Atoi(ev.ExtraParam)
	if err != nil || amount <= 0 {
		p.emitImpossible(ev)
		return
	}

	ps.HP += amount
	if ps.HP > 100 {
		ps.HP = 100
	}
	p.emit(ev.Time, fmt.Sprintf("Player [%d] has restored [%d] of health", ev.PlayerID, amount))
}

func (p *Processor) handleReceiveDamage(ev models.RawEvent) {
	ps := p.getOrCreate(ev.PlayerID)
	if !isActive(ps) {
		p.emitImpossible(ev)
		return
	}

	amount, err := strconv.Atoi(ev.ExtraParam)
	if err != nil || amount <= 0 {
		p.emitImpossible(ev)
		return
	}

	ps.HP -= amount
	p.emit(ev.Time, fmt.Sprintf("Player [%d] recieved [%d] of damage", ev.PlayerID, amount))

	if ps.HP <= 0 {
		ps.HP = 0
		ps.Dead = true
		ps.Finished = true
		ps.DungeonEndTime = ev.Time
		ps.FinalHP = 0
		ps.FinalStatus = models.StatusFail
		ps.FinalTime = ev.Time.Sub(ps.DungeonEnterTime)
		p.emit(ev.Time, fmt.Sprintf("Player [%d] is dead", ev.PlayerID))
	}
}
