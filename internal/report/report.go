package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"dungeon-event-processor/internal/models"
)

// Generate builds the final report string from player states.
func Generate(players map[int]*models.PlayerState) string {
	// Sort players by ID for deterministic output.
	ids := make([]int, 0, len(players))
	for id := range players {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	var sb strings.Builder
	sb.WriteString("Final report:\n")

	for _, id := range ids {
		ps := players[id]
		sb.WriteString(formatPlayer(ps))
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func formatPlayer(ps *models.PlayerState) string {
	totalTime := formatDuration(ps.FinalTime)

	// Average floor clear time (boss floor not included).
	avgFloor := zeroDuration()
	if len(ps.FloorTimes) > 0 {
		var sum time.Duration
		for _, d := range ps.FloorTimes {
			sum += d
		}
		avg := sum / time.Duration(len(ps.FloorTimes))
		avgFloor = formatDuration(avg)
	}

	bossTime := formatDuration(ps.BossKillTime)
	hp := ps.FinalHP

	return fmt.Sprintf("[%s] %d [%s, %s, %s] HP:%d",
		ps.FinalStatus, ps.ID, totalTime, avgFloor, bossTime, hp)
}

// formatDuration formats a duration as HH:MM:SS with trailing zeros.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func zeroDuration() string {
	return "00:00:00"
}