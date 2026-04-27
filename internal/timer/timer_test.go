package timer

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{name: "minutes", value: "30m", want: 30 * time.Minute},
		{name: "hours", value: "2h", want: 2 * time.Hour},
		{name: "hours and minutes", value: "1h30m", want: 90 * time.Minute},
		{name: "uppercase", value: "1H15M", want: 75 * time.Minute},
		{name: "empty", value: "", wantErr: true},
		{name: "zero", value: "0m", wantErr: true},
		{name: "missing unit", value: "30", wantErr: true},
		{name: "unsupported seconds", value: "30s", wantErr: true},
		{name: "minutes before hours", value: "30m1h", wantErr: true},
		{name: "repeated hours", value: "1h2h", wantErr: true},
		{name: "repeated minutes", value: "10m20m", wantErr: true},
		{name: "invalid", value: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseUntil(t *testing.T) {
	location := time.FixedZone("test", 7*60*60)
	now := time.Date(2026, 4, 24, 14, 0, 0, 0, location)

	tests := []struct {
		name    string
		now     time.Time
		value   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "later today",
			now:   now,
			value: "17:30",
			want:  time.Date(2026, 4, 24, 17, 30, 0, 0, location),
		},
		{
			name:  "tomorrow when time already passed",
			now:   time.Date(2026, 4, 24, 22, 0, 0, 0, location),
			value: "17:30",
			want:  time.Date(2026, 4, 25, 17, 30, 0, 0, location),
		},
		{
			name:  "single digit hour",
			now:   now,
			value: "9:15",
			want:  time.Date(2026, 4, 25, 9, 15, 0, 0, location),
		},
		{name: "invalid hour", now: now, value: "99:15", wantErr: true},
		{name: "invalid minute", now: now, value: "12:99", wantErr: true},
		{name: "missing colon", now: now, value: "1730", wantErr: true},
		{name: "short minute", now: now, value: "17:3", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUntil(tt.now, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
