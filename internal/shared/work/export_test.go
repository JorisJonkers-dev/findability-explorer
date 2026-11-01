package work

import "time"

// SetBeatEvery makes running work beat every d, and returns what puts the interval back.
func SetBeatEvery(d time.Duration) (restore func()) {
	old := beatEvery
	beatEvery = d
	return func() { beatEvery = old }
}
