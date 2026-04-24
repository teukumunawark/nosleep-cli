package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"nosleep-cli/internal/keepawake"
	"nosleep-cli/internal/tui"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "NoSleep CLI - Windows sleep prevention utility\n\n")
		fmt.Fprintf(os.Stderr, "Keeps the system and display awake without simulating mouse or keyboard input.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  nosleep.exe [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  nosleep.exe -duration 45m -mode \"Monitoring\"\n")
		fmt.Fprintf(os.Stderr, "  nosleep.exe -duration 2h\n\n")
		fmt.Fprintf(os.Stderr, "Notes:\n")
		fmt.Fprintf(os.Stderr, "  - Press q, esc, or Ctrl+C to stop the session.\n")
		fmt.Fprintf(os.Stderr, "  - Uses the Windows SetThreadExecutionState API.\n")
	}

	durationStr := flag.String("duration", "0", "Session duration, for example 30m or 1h. Use 0 to run until stopped.")
	mode := flag.String("mode", "generic", "Optional session label, for example Monitoring or Reading.")
	flag.Parse()

	duration, err := time.ParseDuration(*durationStr)
	if err != nil {
		fmt.Printf("Error: invalid duration %q.\n", *durationStr)
		os.Exit(1)
	}

	_ = keepawake.SetKeepAwake(true)

	if err := tui.Start(duration, *mode); err != nil {
		fmt.Printf("Error: %v\n", err)
		_ = keepawake.SetKeepAwake(false)
		os.Exit(1)
	}

	_ = keepawake.SetKeepAwake(false)
}
