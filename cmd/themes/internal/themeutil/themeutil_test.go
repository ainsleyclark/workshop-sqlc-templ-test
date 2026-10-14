package themeutil

import (
	"testing"
	"time"
	"workshop/internal/domain/sighting"
	"workshop/internal/domain/species"

	"github.com/stretchr/testify/assert"
)

func TestDisplayName(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input species.Species
		want  string
	}{
		"Prefers the vernacular name": {species.Species{VernacularName: "humpback whale", CanonicalName: "Megaptera novaeangliae"}, "humpback whale"},
		"Falls back to canonical":     {species.Species{CanonicalName: "Megaptera novaeangliae"}, "Megaptera novaeangliae"},
		"Falls back to scientific":    {species.Species{ScientificName: "Megaptera novaeangliae (Borowski, 1781)"}, "Megaptera novaeangliae (Borowski, 1781)"},
		"Nothing at all":              {species.Species{}, "Unknown species"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, DisplayName(test.input))
		})
	}
}

func TestPlace(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input sighting.Location
		want  string
	}{
		"Every part":        {sighting.Location{Locality: "Kwale", StateProvince: "Coast", Country: "Kenya"}, "Kwale, Coast, Kenya"},
		"Gaps are left out": {sighting.Location{Locality: "Kwale", Country: "Kenya"}, "Kwale, Kenya"},
		"Country only":      {sighting.Location{Country: "Kenya"}, "Kenya"},
		"Nothing":           {sighting.Location{}, "Location unknown"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, Place(test.input))
		})
	}
}

func TestCoordinates(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input sighting.Location
		want  string
	}{
		"North east":  {sighting.Location{Coordinates: sighting.Coordinates{Latitude: 51.5072, Longitude: 0.1276}}, "51.507N, 0.128E"},
		"South west":  {sighting.Location{Coordinates: sighting.Coordinates{Latitude: -4.6583, Longitude: -39.2083}}, "4.658S, 39.208W"},
		"Null island": {sighting.Location{}, "0.000N, 0.000E"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, Coordinates(test.input))
		})
	}
}

func TestIndividuals(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input *int
		want  string
	}{
		"Not recorded": {nil, ""},
		"One":          {new(1), "1 individual"},
		"Several":      {new(4), "4 individuals"},
		"Zero":         {new(0), "0 individuals"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, Individuals(test.input))
		})
	}
}

func TestPercent(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		count, total int
		want         int
	}{
		"Half":                           {5, 10, 50},
		"All":                            {10, 10, 100},
		"None":                           {0, 10, 0},
		"Rounds to nearest":              {1, 3, 33},
		"A zero total cannot divide":     {5, 0, 0},
		"A negative total cannot divide": {5, -1, 0},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, Percent(test.count, test.total))
		})
	}
}

func TestSpan(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input sighting.Summary
		want  string
	}{
		"No records": {sighting.Summary{}, "No records"},
		"One year": {sighting.Summary{
			Sightings: 2,
			Earliest:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Latest:    time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		}, "2025"},
		"Several years": {sighting.Summary{
			Sightings: 2,
			Earliest:  time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
			Latest:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}, "2015–2026"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, Span(test.input))
		})
	}
}

func TestYears(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input sighting.Summary
		want  []int
	}{
		"No records": {sighting.Summary{}, nil},
		"Keeps the years in between": {sighting.Summary{
			Sightings: 2,
			Earliest:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Latest:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}, []int{2023, 2024, 2025, 2026}},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, Years(test.input))
		})
	}
}

func TestMapPoints(t *testing.T) {
	t.Parallel()

	t.Run("Skips records with no position", func(t *testing.T) {
		t.Parallel()
		got := MapPoints([]sighting.Sighting{
			{Location: sighting.Location{Coordinates: sighting.Coordinates{Latitude: 0, Longitude: 0}}},
			{Location: sighting.Location{Coordinates: sighting.Coordinates{Latitude: -4.65, Longitude: 39.2}, Country: "Kenya"}},
		})
		assert.Len(t, got, 1)
		assert.Equal(t, "Kenya", got[0].Place)
	})

	t.Run("Carries the anchor of its record", func(t *testing.T) {
		t.Parallel()
		s := sighting.Sighting{Location: sighting.Location{Coordinates: sighting.Coordinates{Latitude: 1, Longitude: 2}}}
		got := MapPoints([]sighting.Sighting{s})
		assert.Equal(t, Anchor(s), got[0].Anchor)
	})
}

func TestSince(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

	tt := map[string]struct {
		input time.Time
		want  string
	}{
		"Today":         {now, "Today"},
		"Yesterday":     {now.AddDate(0, 0, -1), "Yesterday"},
		"Days":          {now.AddDate(0, 0, -5), "5 days ago"},
		"Months":        {now.AddDate(0, 0, -90), "3 months ago"},
		"A year":        {now.AddDate(0, 0, -400), "A year ago"},
		"Years":         {now.AddDate(0, 0, -900), "2 years ago"},
		"In the future": {now.AddDate(0, 0, 5), "Upcoming"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, Since(test.input, now))
		})
	}
}
