package kenya

import (
	"cmp"
	"strconv"
	"strings"
	"time"
	"workshop/cmd/themes/internal/themeutil"
	"workshop/internal/domain/sighting"
)

// Props is everything the issue renders.
type Props struct {
	Entries []Entry
	Summary sighting.Summary
	Species []sighting.Tally
	Coast   []sighting.Tally
	Points  []themeutil.MapPoint
	Now     time.Time
}

// Entry is one sighting written up as a numbered article.
type Entry struct {
	Sighting sighting.Sighting
	Number   int
}

// NewProps arranges the sightings into an issue, newest first.
func NewProps(sightings []sighting.Sighting, now time.Time) Props {
	entries := make([]Entry, len(sightings))
	for i, s := range sightings {
		entries[i] = Entry{Sighting: s, Number: i + 1}
	}

	coast := make(map[string]int)
	for _, s := range sightings {
		if name := s.Location.StateProvince; name != "" {
			coast[name]++
		}
	}

	return Props{
		Entries: entries,
		Summary: sighting.Summarise(sightings),
		Species: sighting.Top(sighting.CountBy(sightings, speciesName), 5),
		Coast:   sighting.Top(coast, 5),
		Points:  themeutil.MapPoints(sightings),
		Now:     now,
	}
}

// folio renders an entry's number the way a magazine would.
func folio(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// count renders a tally for the templates.
func count(n int) string {
	return strconv.Itoa(n)
}

// standfirst is the sentence under the headline, which changes with how
// much there is to report.
func standfirst(s sighting.Summary) string {
	switch s.Sightings {
	case 0:
		return "No sightings have been recorded in Kenyan waters yet."
	case 1:
		return "A single record from the Kenyan coast, in full."
	}
	return "Every cetacean sighting logged along the Kenyan coast, read end to end."
}

// basis turns GBIF's shouted enum into something a reader can take in.
func basis(s sighting.Sighting) string {
	if s.BasisOfRecord == "" {
		return "Not stated"
	}
	words := strings.ReplaceAll(strings.ToLower(s.BasisOfRecord), "_", " ")
	return strings.ToUpper(words[:1]) + words[1:]
}

// credit names whoever took the first photograph on a sighting, with
// the licence it was published under where GBIF supplied one.
func credit(s sighting.Sighting) string {
	for _, m := range s.Media {
		if m.Type != "StillImage" || m.URL == "" {
			continue
		}
		name := cmp.Or(m.Creator, m.RightsHolder, "Unattributed")
		if m.License != "" {
			return name + " · " + licence(m.License)
		}
		return name
	}
	return ""
}

// licence shortens the licence URLs GBIF returns to their name.
func licence(raw string) string {
	switch {
	case strings.Contains(raw, "by-nc-sa"):
		return "CC BY-NC-SA"
	case strings.Contains(raw, "by-nc-nd"):
		return "CC BY-NC-ND"
	case strings.Contains(raw, "by-nc"):
		return "CC BY-NC"
	case strings.Contains(raw, "by-sa"):
		return "CC BY-SA"
	case strings.Contains(raw, "by-nd"):
		return "CC BY-ND"
	case strings.Contains(raw, "publicdomain"), strings.Contains(raw, "zero"):
		return "CC0"
	case strings.Contains(raw, "creativecommons.org/licenses/by"):
		return "CC BY"
	}
	return raw
}

// speciesName labels a tally row with the name this theme shows.
func speciesName(s sighting.Sighting) string {
	return themeutil.DisplayName(s.Species)
}
