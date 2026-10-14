package antarctica

import (
	"strconv"
	"time"
	"workshop/cmd/themes/internal/themeutil"
	"workshop/internal/domain/sighting"
)

// Props is everything the station page renders. It is built once in
// NewProps so the templates only read.
type Props struct {
	Sightings  []sighting.Sighting
	Summary    sighting.Summary
	Species    []sighting.Tally
	Localities []sighting.Tally
	Observers  []sighting.Tally
	Points     []themeutil.MapPoint
	Now        time.Time
}

// NewProps reduces the sightings to the readouts the station shows.
func NewProps(sightings []sighting.Sighting, now time.Time) Props {
	localities := make(map[string]int)
	observers := make(map[string]int)
	for _, s := range sightings {
		if s.Location.Locality != "" {
			localities[s.Location.Locality]++
		}
		if s.RecordedBy != "" {
			observers[s.RecordedBy]++
		}
	}

	return Props{
		Sightings:  sightings,
		Summary:    sighting.Summarise(sightings),
		Species:    sighting.Top(sighting.CountBy(sightings, speciesName), 6),
		Localities: sighting.Top(localities, 8),
		Observers:  sighting.Top(observers, 6),
		Points:     themeutil.MapPoints(sightings),
		Now:        now,
	}
}

// bar scales a row against the longest row beside it and hands CSS the
// width as a custom property.
func bar(rows []sighting.Tally, count int) map[string]string {
	highest := 0
	for _, r := range rows {
		if r.Count > highest {
			highest = r.Count
		}
	}
	return map[string]string{"--bar": strconv.Itoa(themeutil.Percent(count, highest)) + "%"}
}

// count renders a tally, which templates cannot convert themselves.
func count(n int) string {
	return strconv.Itoa(n)
}

// speciesName labels a tally row with the name this theme shows.
func speciesName(s sighting.Sighting) string {
	return themeutil.DisplayName(s.Species)
}
