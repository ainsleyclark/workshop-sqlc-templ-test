package humpback

import (
	"fmt"
	"strconv"
	"time"
	"workshop/cmd/themes/internal/themeutil"
	"workshop/internal/domain/sighting"
)

// galleryLimit caps the picture grid. The ledger below it still lists
// every record, so nothing is hidden by the cap, but the page does not
// ask the browser for hundreds of photographs at once.
const galleryLimit = 48

type (
	// Props is everything the console renders.
	Props struct {
		Sightings []sighting.Sighting
		Gallery   []sighting.Sighting
		Summary   sighting.Summary
		Years     []YearBar
		Countries []sighting.Tally
		Observers []sighting.Tally
		Basis     []sighting.Tally
		Points    []themeutil.MapPoint
		Now       time.Time
	}
	// YearBar is one column of the records-per-year chart.
	YearBar struct {
		Year    int
		Count   int
		Percent int
	}
)

// NewProps reduces the sightings to the panels the console shows.
func NewProps(sightings []sighting.Sighting, now time.Time) Props {
	summary := sighting.Summarise(sightings)

	observers := make(map[string]int)
	basis := make(map[string]int)
	for _, s := range sightings {
		if s.RecordedBy != "" {
			observers[s.RecordedBy]++
		}
		if s.BasisOfRecord != "" {
			basis[label(s.BasisOfRecord)]++
		}
	}

	gallery := sightings
	if len(gallery) > galleryLimit {
		gallery = gallery[:galleryLimit]
	}

	return Props{
		Sightings: sightings,
		Gallery:   gallery,
		Summary:   summary,
		Years:     years(sightings, summary),
		Countries: sighting.Top(sighting.CountBy(sightings, country), 8),
		Observers: sighting.Top(observers, 6),
		Basis:     sighting.Top(basis, 4),
		Points:    themeutil.MapPoints(sightings),
		Now:       now,
	}
}

// years builds the histogram, keeping empty years so a gap in the
// record reads as a gap rather than being closed up.
func years(sightings []sighting.Sighting, summary sighting.Summary) []YearBar {
	counts := sighting.CountByYear(sightings)

	highest := 0
	for _, c := range counts {
		if c > highest {
			highest = c
		}
	}

	all := themeutil.Years(summary)
	bars := make([]YearBar, len(all))
	for i, y := range all {
		bars[i] = YearBar{
			Year:    y,
			Count:   counts[y],
			Percent: themeutil.Percent(counts[y], highest),
		}
	}

	return bars
}

// label turns GBIF's shouted enum into something a reader can take in.
func label(raw string) string {
	switch raw {
	case "HUMAN_OBSERVATION":
		return "Human observation"
	case "MACHINE_OBSERVATION":
		return "Machine observation"
	case "PRESERVED_SPECIMEN":
		return "Preserved specimen"
	case "MATERIAL_SAMPLE":
		return "Material sample"
	case "OCCURRENCE":
		return "Occurrence"
	}
	return raw
}

// count renders a tally for the templates.
func count(n int) string {
	return strconv.Itoa(n)
}

// height hands a chart column its share of the plot as a percentage.
func height(pct int) map[string]string {
	return map[string]string{"--bar": strconv.Itoa(pct) + "%"}
}

// width scales a leaderboard row against the longest row beside it.
func width(rows []sighting.Tally, n int) map[string]string {
	highest := 0
	for _, r := range rows {
		if r.Count > highest {
			highest = r.Count
		}
	}
	return map[string]string{"--bar": strconv.Itoa(themeutil.Percent(n, highest)) + "%"}
}

// share renders a row's share of the whole, for the text beside a bar.
func share(n, total int) string {
	return strconv.Itoa(themeutil.Percent(n, total)) + "%"
}

// shortYear renders a year for a cramped axis, so the labels do not
// collide on a narrow screen.
func shortYear(y int) string {
	return fmt.Sprintf("'%02d", y%100)
}

// country labels a tally row with the country a sighting was made in.
func country(s sighting.Sighting) string {
	return s.Location.Country
}
