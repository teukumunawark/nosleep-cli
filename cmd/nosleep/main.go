package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"nosleep-cli/internal/keepawake"
	"nosleep-cli/internal/tui"
)

func usagePrintf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

func main() {
	flag.Usage = func() {
		usagePrintf("NoSleep CLI - Windows sleep prevention utility\n\n")
		usagePrintf("Keeps the system and display awake without simulating mouse or keyboard input.\n\n")
		usagePrintf("Usage:\n")
		usagePrintf("  nosleep.exe [flags]\n\n")
		usagePrintf("Flags:\n")
		flag.PrintDefaults()
		usagePrintf("\nExamples:\n")
		usagePrintf("  nosleep.exe -duration 45m -mode \"Monitoring\"\n")
		usagePrintf("  nosleep.exe -duration 2h\n\n")
		usagePrintf("Notes:\n")
		usagePrintf("  - Press q, esc, or Ctrl+C to stop the session.\n")
		usagePrintf("  - Uses the Windows SetThreadExecutionState API.\n")
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
