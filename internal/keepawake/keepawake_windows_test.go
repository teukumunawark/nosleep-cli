package keepawake

import (
	"testing"
)

func TestSetKeepAwake(t *testing.T) {
	err := SetKeepAwake(true)
	if err != nil {
		t.Fatalf("failed to set keep awake to true: %v", err)
	}

	err = SetKeepAwake(false)
	if err != nil {
		t.Fatalf("failed to set keep awake to false: %v", err)
	}
}
