package app

import "time"

// SetRecBeat makes a running Recommendation run beat every d.
func (s *Service) SetRecBeat(d time.Duration) { s.writing.beat = d }
