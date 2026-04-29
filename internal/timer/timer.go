package timer

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const maxDuration = time.Duration(1<<63 - 1)

func ParseDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("duration is required")
	}

	var total time.Duration
	var number strings.Builder
	seenUnit := false
	seenHours := false
	seenMinutes := false

	for _, r := range value {
		switch {
		case unicode.IsDigit(r):
			number.WriteRune(r)
		case r == 'h' || r == 'H' || r == 'm' || r == 'M':
			if number.Len() == 0 {
				return 0, fmt.Errorf("missing number before %q", r)
			}

			n, err := strconv.ParseInt(number.String(), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse duration number: %w", err)
			}
			number.Reset()

			switch r {
			case 'h', 'H':
				if seenHours || seenMinutes {
					return 0, fmt.Errorf("hours must appear once before minutes")
				}
				total, err = addDurationPart(total, n, time.Hour)
				if err != nil {
					return 0, err
				}
				seenHours = true
			case 'm', 'M':
				if seenMinutes {
					return 0, fmt.Errorf("minutes must appear once")
				}
				total, err = addDurationPart(total, n, time.Minute)
				if err != nil {
					return 0, err
				}
				seenMinutes = true
			}
			seenUnit = true
		default:
			return 0, fmt.Errorf("unsupported duration character %q", r)
		}
	}

	if number.Len() > 0 {
		return 0, fmt.Errorf("missing duration unit")
	}
	if !seenUnit || total <= 0 {
		return 0, fmt.Errorf("duration must be greater than zero")
	}

	return total, nil
}

func addDurationPart(total time.Duration, n int64, unit time.Duration) (time.Duration, error) {
	if n > int64(maxDuration/unit) {
		return 0, fmt.Errorf("duration is too large")
	}

	part := time.Duration(n) * unit
	if total > maxDuration-part {
		return 0, fmt.Errorf("duration is too large")
	}

	return total + part, nil
}

func ParseUntil(now time.Time, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("time must use HH:MM format")
	}

	hour, err := parseClockPart(parts[0], 1, 2)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour: %w", err)
	}
	minute, err := parseClockPart(parts[1], 2, 2)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute: %w", err)
	}
	if hour > 23 {
		return time.Time{}, fmt.Errorf("hour must be between 0 and 23")
	}
	if minute > 59 {
		return time.Time{}, fmt.Errorf("minute must be between 0 and 59")
	}

	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !target.After(now) {
		target = target.AddDate(0, 0, 1)
	}

	return target, nil
}

func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func SameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func parseClockPart(value string, minLen, maxLen int) (int, error) {
	if len(value) < minLen || len(value) > maxLen {
		return 0, fmt.Errorf("expected %d-%d digits", minLen, maxLen)
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return 0, fmt.Errorf("contains non-digit %q", r)
		}
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return n, nil
}
