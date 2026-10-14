package sighting

import (
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
)

func TestCreateParams_Validate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)

	valid := CreateParams{
		SpeciesID:  uuid.New(),
		HappenedAt: now.Add(-time.Hour),
		Location:   Location{Coordinates: Coordinates{Latitude: 51.5, Longitude: -0.12}},
	}

	tt := map[string]struct {
		mutate   func(p *CreateParams)
		wantErrs []string
	}{
		"Valid": {
			mutate: func(*CreateParams) {},
		},
		"Boundary coordinates": {
			mutate: func(p *CreateParams) { p.Location.Coordinates = Coordinates{Latitude: -90, Longitude: 180} },
		},
		"Happened now": {
			mutate: func(p *CreateParams) { p.HappenedAt = now },
		},
		"Zero individual count": {
			mutate: func(p *CreateParams) { p.IndividualCount = new(0) },
		},
		"Missing species ID": {
			mutate:   func(p *CreateParams) { p.SpeciesID = uuid.Nil() },
			wantErrs: []string{"species ID is required"},
		},
		"Missing happened at": {
			mutate:   func(p *CreateParams) { p.HappenedAt = time.Time{} },
			wantErrs: []string{"happened at is required"},
		},
		"Happened in the future": {
			mutate:   func(p *CreateParams) { p.HappenedAt = now.Add(time.Second) },
			wantErrs: []string{"happened at cannot be in the future"},
		},
		"Latitude too low": {
			mutate:   func(p *CreateParams) { p.Location.Latitude = -90.1 },
			wantErrs: []string{"latitude must be between -90 and 90"},
		},
		"Latitude too high": {
			mutate:   func(p *CreateParams) { p.Location.Latitude = 90.1 },
			wantErrs: []string{"latitude must be between -90 and 90"},
		},
		"Longitude too low": {
			mutate:   func(p *CreateParams) { p.Location.Longitude = -180.1 },
			wantErrs: []string{"longitude must be between -180 and 180"},
		},
		"Longitude too high": {
			mutate:   func(p *CreateParams) { p.Location.Longitude = 180.1 },
			wantErrs: []string{"longitude must be between -180 and 180"},
		},
		"Negative count": {
			mutate:   func(p *CreateParams) { p.IndividualCount = new(-1) },
			wantErrs: []string{"individual count cannot be negative"},
		},
		"Reports every broken rule": {
			mutate: func(p *CreateParams) {
				p.SpeciesID = uuid.Nil()
				p.Location.Latitude = 100
			},
			wantErrs: []string{
				"species ID is required",
				"latitude must be between -90 and 90",
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			in := valid
			test.mutate(&in)

			err := in.Validate(now)
			assert.Equal(t, len(test.wantErrs) > 0, errors.Is(err, ErrInvalid))
			for _, want := range test.wantErrs {
				assert.ErrorContains(t, err, want)
			}
		})
	}
}
