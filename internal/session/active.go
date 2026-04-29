package session

import (
	"errors"
	"time"
)

func ActiveState(store Store) (State, bool, error) {
	state, ok, err := store.Read()
	if err != nil {
		if errors.Is(err, ErrInvalidState) {
			if removeErr := store.Remove(); removeErr != nil {
				return State{}, false, removeErr
			}
			return State{}, false, nil
		}
		return State{}, false, err
	}
	if !ok {
		return State{}, false, nil
	}

	matches, err := ProcessMatches(state.PID, state.Executable, state.ProcessStartedAt)
	if err != nil {
		return State{}, false, err
	}
	if !matches {
		if err := store.Remove(); err != nil {
			return State{}, false, err
		}
		return State{}, false, nil
	}
	if state.AutoStopAt != nil && !state.AutoStopAt.After(time.Now()) {
		if err := store.Remove(); err != nil {
			return State{}, false, err
		}
		return State{}, false, nil
	}

	return state, true, nil
}
