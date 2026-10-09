package sighting

import (
	"errors"
	"fmt"
	"time"
)

// Validate checks the sighting against the rules every sighting must
// follow. Every broken rule is returned, each wrapping ErrInvalid.
func (s Sighting) Validate(now time.Time) error {
	var errs []error

	if s.Key == 0 {
		errs = append(errs, fmt.Errorf("%w: key is required", ErrInvalid))
	}
	if s.Species.Key == 0 {
		errs = append(errs, fmt.Errorf("%w: species is required", ErrInvalid))
	}
	if s.Latitude < -90 || s.Latitude > 90 {
		errs = append(errs, fmt.Errorf("%w: latitude must be between -90 and 90", ErrInvalid))
	}
	if s.Longitude < -180 || s.Longitude > 180 {
		errs = append(errs, fmt.Errorf("%w: longitude must be between -180 and 180", ErrInvalid))
	}
	if s.EventDate == "" {
		errs = append(errs, fmt.Errorf("%w: event date is required", ErrInvalid))
	}
	// EventDate comes in three formats, but each starts with the day,
	// so the day is all this rule can safely read.
	if day, err := time.Parse(time.DateOnly, s.EventDate[:min(len(s.EventDate), len(time.DateOnly))]); err == nil && day.After(now) {
		errs = append(errs, fmt.Errorf("%w: event date cannot be in the future", ErrInvalid))
	}

	return errors.Join(errs...)
}
