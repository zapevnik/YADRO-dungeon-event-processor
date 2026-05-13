package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"dungeon-event-processor/internal/models"
)

// LoadEvents opens file and parses all events.
func LoadEvents(path string) ([]models.RawEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open events file: %w", err)
	}
	defer f.Close()

	return parseEvents(f)
}

// parseEvents reads all events from reader and returns them in order.
func parseEvents(r io.Reader) ([]models.RawEvent, error) {
	var result []models.RawEvent

	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ev, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		result = append(result, ev)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// parseLine parses a single event line:
// [HH:MM:SS] <playerID> <eventID> [extraParam...]
func parseLine(line string) (models.RawEvent, error) {
	if len(line) < 10 || line[0] != '[' {
		return models.RawEvent{}, fmt.Errorf("invalid line: %q", line)
	}

	closeIdx := strings.Index(line, "]")
	if closeIdx < 0 {
		return models.RawEvent{}, fmt.Errorf("missing ']' in: %q", line)
	}

	timeStr := line[1:closeIdx]

	t, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		return models.RawEvent{}, fmt.Errorf("invalid time %q: %w", timeStr, err)
	}

	rest := strings.TrimSpace(line[closeIdx+1:])
	parts := strings.Fields(rest)

	if len(parts) < 2 {
		return models.RawEvent{}, fmt.Errorf("too few fields in: %q", line)
	}

	playerID, err := strconv.Atoi(parts[0])
	if err != nil {
		return models.RawEvent{}, fmt.Errorf("invalid playerID %q", parts[0])
	}

	eventID, err := strconv.Atoi(parts[1])
	if err != nil {
		return models.RawEvent{}, fmt.Errorf("invalid eventID %q", parts[1])
	}

	extraParam := ""
	if len(parts) > 2 {
		extraParam = strings.Join(parts[2:], " ")
	}

	return models.RawEvent{
		Time:       t,
		PlayerID:   playerID,
		EventID:    eventID,
		ExtraParam: extraParam,
	}, nil
}