package main

import (
	"fmt"
	"os"
	"strings"

	"dungeon-event-processor/internal/parser"
	"dungeon-event-processor/internal/processor"
	"dungeon-event-processor/internal/report"
)

func main() {
	cfg, err := parser.LoadCfg("task_data/config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	rawEvents, err := parser.LoadEvents("task_data/events")
	if err != nil {
		fmt.Println(err)
		return
	}

	proc := processor.New(cfg)
	outputLines := proc.Process(rawEvents)

	// Print event log.
	var sb strings.Builder
	for _, line := range outputLines {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", line.Time.Format("15:04:05"), line.Message))
	}
	fmt.Print(sb.String())

	// Print final report.
	fmt.Println()
	fmt.Println(report.Generate(proc.Players()))
}