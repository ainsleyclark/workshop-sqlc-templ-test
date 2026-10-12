package sighting_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"
	"workshop/internal/clients/gbif"
	"workshop/internal/clients/openmeteo"
	"workshop/internal/domain/sighting"
	"workshop/internal/domain/sighting/country"
	"workshop/internal/domain/species"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGBIF answers ingest from memory, so the domain's tests are written
// in the client's Go types rather than GBIF's JSON.
type fakeGBIF struct {
	// occurrences is everything the search matches, served a page at a time.
	occurrences []gbif.Occurrence
	// pageSize caps each page, as GBIF caps it at 300. Zero means no cap.
	pageSize int
	// err fails every search; speciesErr fails every species lookup.
	err        error
	speciesErr error

	// searches and lookups record what ingest asked for.
	searches []gbif.OccurrencesSearchArgs
	lookups  int
}

func (f *fakeGBIF) Occurrences(_ context.Context, args gbif.OccurrencesSearchArgs) (gbif.PaginatedResponse[gbif.Occurrence], error) {
	f.searches = append(f.searches, args)
	if f.err != nil {
		return gbif.PaginatedResponse[gbif.Occurrence]{}, f.err
	}

	start := min(args.Offset, len(f.occurrences))
	end := len(f.occurrences)
	if args.Limit > 0 {
		end = min(start+args.Limit, end)
	}
	if f.pageSize > 0 {
		end = min(start+f.pageSize, end)
	}

	return gbif.PaginatedResponse[gbif.Occurrence]{
		Offset:       args.Offset,
		Limit:        args.Limit,
		EndOfRecords: end == len(f.occurrences),
		Count:        int64(len(f.occurrences)),
		Results:      f.occurrences[start:end],
	}, nil
}

// Species returns a lion under whichever key is asked for, so each new
// key saves a species of its own.
func (f *fakeGBIF) Species(_ context.Context, key int) (gbif.Species, error) {
	f.lookups++
	if f.speciesErr != nil {
		return gbif.Species{}, f.speciesErr
	}

	lion := lionParams()
	return gbif.Species{
		Key:            key,
		Kingdom:        lion.Kingdom,
		Phylum:         lion.Phylum,
		Class:          lion.Class,
		Order:          lion.Order,
		Family:         lion.Family,
		Genus:          lion.Genus,
		Species:        lion.Species,
		Rank:           lion.Rank,
		ScientificName: lion.ScientificName,
		CanonicalName:  lion.CanonicalName,
		VernacularName: lion.VernacularName,
	}, nil
}

// fakeOpenMeteo answers every request with the same hour of weather,
// placed where and when it was asked for.
type fakeOpenMeteo struct {
	reading openmeteo.Reading
	err     error
}

func (f *fakeOpenMeteo) Reading(_ context.Context, req openmeteo.Request) (openmeteo.Reading, error) {
	if f.err != nil {
		return openmeteo.Reading{}, f.err
	}

	r := f.reading
	r.Coordinates = req.Coordinates
	r.Time = req.Time.UTC().Truncate(time.Hour)

	return r, nil
}

// settledReading is the weather fakeOpenMeteo reports by default.
var settledReading = openmeteo.Reading{
	Temperature:         26.3,
	ApparentTemperature: 28.1,
	Humidity:            74,
	CloudCover:          22,
	WindSpeed:           14.2,
	WindGusts:           27.4,
	WindDirection:       118,
	Code:                openmeteo.WeatherCode(1),
}

// newOccurrence returns a fully populated lion occurrence in the Maasai Mara.
func newOccurrence(key int64) gbif.Occurrence {
	return gbif.Occurrence{
		Key:                           key,
		OccurrenceID:                  fmt.Sprintf("occurrence-%d", key),
		SpeciesKey:                    lionKey,
		EventDate:                     happenedAt,
		BasisOfRecord:                 "HUMAN_OBSERVATION",
		RecordedBy:                    "Ada|Grace",
		IndividualCount:               new(2),
		OccurrenceRemarks:             "Resting in the shade",
		References:                    "https://www.inaturalist.org/observations/1",
		Country:                       "Kenya",
		CountryCode:                   "KE",
		StateProvince:                 "Narok",
		Locality:                      "Maasai Mara",
		DecimalLatitude:               -1.5,
		DecimalLongitude:              35.1,
		CoordinateUncertaintyInMeters: new(10.0),
		Elevation:                     new(1600.0),
	}
}

// The one branch left uncovered is saving a species GBIF has just
// described, which needs a store that reads but won't write.
func TestService_Ingest(t *testing.T) {
	type Test struct {
		ctx       context.Context
		svc       *sighting.Service
		species   *species.Service
		gbif      *fakeGBIF
		openMeteo *fakeOpenMeteo
	}

	setupIngest := func(t *testing.T, fail bool) Test {
		t.Helper()

		test := Test{
			gbif:      &fakeGBIF{},
			openMeteo: &fakeOpenMeteo{reading: settledReading},
		}
		test.ctx, test.svc, test.species = setup(t, fail, test.gbif, test.openMeteo)

		return test
	}

	t.Run("Creates a sighting per occurrence", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1), newOccurrence(2)}

		got, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Len(t, got.Created, 2)
		assert.Equal(t, 2, got.Fetched)
		assert.Zero(t, got.Skipped+got.Duplicates+got.Failed)
	})

	t.Run("Maps the occurrence onto the sighting", func(t *testing.T) {
		test := setupIngest(t, false)
		occ := newOccurrence(1)
		test.gbif.occurrences = []gbif.Occurrence{occ}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		require.Len(t, res.Created, 1)

		got := res.Created[0]
		assert.Equal(t, sighting.CreateParams{
			GBIFKey:         occ.Key,
			OccurrenceID:    occ.OccurrenceID,
			HappenedAt:      occ.EventDate,
			BasisOfRecord:   occ.BasisOfRecord,
			RecordedBy:      occ.RecordedBy,
			IndividualCount: occ.IndividualCount,
			Remarks:         occ.OccurrenceRemarks,
			ReferenceURL:    occ.References,
			Location: sighting.Location{
				Country:                       occ.Country,
				CountryCode:                   country.Kenya,
				StateProvince:                 occ.StateProvince,
				Locality:                      occ.Locality,
				Coordinates:                   sighting.Coordinates{Latitude: -1.5, Longitude: 35.1},
				CoordinateUncertaintyInMetres: occ.CoordinateUncertaintyInMeters,
				ElevationInMetres:             occ.Elevation,
			},
			SpeciesID: got.Species.ID,
			WeatherID: got.Weather.ID,
			Media:     []sighting.Media{},
		}, got.CreateParams())
	})

	t.Run("Copies the media", func(t *testing.T) {
		test := setupIngest(t, false)
		occ := newOccurrence(1)
		// GBIF's media is an anonymous struct, so grow the slice rather
		// than spell its type out.
		occ.Media = slices.Grow(occ.Media, 1)[:1]
		occ.Media[0].Type = "StillImage"
		occ.Media[0].Format = "image/jpeg"
		occ.Media[0].Identifier = "https://example.com/lion.jpg"
		occ.Media[0].Creator = "Ada"
		occ.Media[0].License = "CC BY 4.0"
		test.gbif.occurrences = []gbif.Occurrence{occ}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		require.Len(t, res.Created, 1)
		require.Len(t, res.Created[0].Media, 1)

		got := res.Created[0].Media[0]
		assert.NotEqual(t, uuid.Nil(), got.ID)
		// The ID is random, so compare everything else.
		got.ID = uuid.Nil()
		assert.Equal(t, sighting.Media{
			Type:    "StillImage",
			Format:  "image/jpeg",
			URL:     "https://example.com/lion.jpg",
			Creator: "Ada",
			License: "CC BY 4.0",
		}, got)
	})

	t.Run("Fetches a new species from GBIF once", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1), newOccurrence(2)}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, test.gbif.lookups)

		got, err := test.svc.Find(test.ctx, res.Created[0].ID)
		require.NoError(t, err)
		assert.Equal(t, lionKey, got.Species.GBIFKey)
	})

	t.Run("Reuses a species already stored", func(t *testing.T) {
		test := setupIngest(t, false)
		lion := seedLion(t, test.ctx, test.species)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1)}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		require.Len(t, res.Created, 1)
		assert.Zero(t, test.gbif.lookups)
		assert.Equal(t, lion.ID, res.Created[0].Species.ID)
	})

	t.Run("Records the weather", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1)}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		require.Len(t, res.Created, 1)

		got, err := test.svc.Find(test.ctx, res.Created[0].ID)
		require.NoError(t, err)
		assert.Equal(t, happenedAt.Truncate(time.Hour), got.Weather.ObservedAt)
		assert.Equal(t, settledReading.Temperature, got.Weather.Temperature.Actual)
		assert.Equal(t, settledReading.WindDirection, got.Weather.Wind.Direction)
		assert.Equal(t, settledReading.Code.String(), got.Weather.Condition.Description)
	})

	t.Run("Builds the search from the request", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1)}
		from := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)

		_, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{
			Limit:       10,
			Offset:      20,
			Country:     country.Kenya,
			Animals:     []species.Animal{species.AnimalLion, species.AnimalCheetah},
			Latitude:    -1.5,
			Longitude:   35.1,
			RadiusMiles: 25,
			DateFrom:    from,
			DateTo:      to,
		})
		require.NoError(t, err)
		require.Len(t, test.gbif.searches, 1)
		assert.Equal(t, gbif.OccurrencesSearchArgs{
			PaginationOptions: gbif.PaginationOptions{Limit: 10, Offset: 20},
			Country:           "KE",
			Latitude:          -1.5,
			Longitude:         35.1,
			RadiusMiles:       25,
			DateFrom:          from,
			DateTo:            to,
			TaxonKeys:         []int{5219404, 2435270},
		}, test.gbif.searches[0])
	})

	t.Run("Pages until the limit", func(t *testing.T) {
		test := setupIngest(t, false)
		for key := range int64(6) {
			test.gbif.occurrences = append(test.gbif.occurrences, newOccurrence(key+1))
		}
		test.gbif.pageSize = 2

		var pages []int
		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{
			Limit:  5,
			OnPage: func(fetched int, _ int64) { pages = append(pages, fetched) },
		})
		require.NoError(t, err)
		assert.Equal(t, 5, res.Fetched)
		assert.Len(t, res.Created, 5)
		assert.Equal(t, []int{2, 4, 5}, pages)
		assert.Equal(t, []gbif.PaginationOptions{
			{Limit: 5, Offset: 0},
			{Limit: 3, Offset: 2},
			{Limit: 1, Offset: 4},
		}, paginationOf(test.gbif.searches))
	})

	t.Run("Stops at the end of records", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1), newOccurrence(2), newOccurrence(3)}
		test.gbif.pageSize = 2

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 3, res.Fetched)
		assert.Len(t, test.gbif.searches, 2)
	})

	t.Run("Skips what a skipper rejects", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1), newOccurrence(2)}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{
			Limit: 10,
			Skippers: []sighting.IngestShouldSkipFn{
				func(o gbif.Occurrence) bool { return o.Key == 2 },
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Skipped)
		require.Len(t, res.Created, 1)
		assert.Equal(t, int64(1), res.Created[0].GBIFKey)
	})

	t.Run("Counts duplicates on re-ingest", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1), newOccurrence(2)}

		_, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, res.Duplicates)
		assert.Empty(t, res.Created)
	})

	t.Run("Counts a failed species lookup", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1)}
		test.gbif.speciesErr = errors.New("species lookup failed")

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Failed)
		assert.Empty(t, res.Created)
	})

	t.Run("Counts a failed weather reading", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1)}
		test.openMeteo.err = errors.New("archive unavailable")

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Failed)
		assert.Empty(t, res.Created)
	})

	t.Run("Counts coordinates out of range", func(t *testing.T) {
		test := setupIngest(t, false)
		occ := newOccurrence(1)
		occ.DecimalLatitude = 100
		test.gbif.occurrences = []gbif.Occurrence{occ}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Failed)
		assert.Empty(t, res.Created)
	})

	t.Run("Counts an unreadable country code", func(t *testing.T) {
		test := setupIngest(t, false)
		occ := newOccurrence(1)
		occ.CountryCode = "KEN"
		test.gbif.occurrences = []gbif.Occurrence{occ}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Failed)
		assert.Empty(t, res.Created)
	})

	t.Run("Counts weather that breaks the rules", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1)}
		test.openMeteo.reading.Humidity = 101

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Failed)
		assert.Empty(t, res.Created)
	})

	t.Run("Counts a sighting that breaks the rules", func(t *testing.T) {
		test := setupIngest(t, false)
		occ := newOccurrence(1)
		occ.EventDate = time.Now().AddDate(0, 0, 1)
		test.gbif.occurrences = []gbif.Occurrence{occ}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Failed)
		assert.Empty(t, res.Created)
	})

	t.Run("Counts a store failure", func(t *testing.T) {
		test := setupIngest(t, true)
		test.gbif.occurrences = []gbif.Occurrence{newOccurrence(1)}

		res, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Failed)
		assert.Zero(t, test.gbif.lookups)
	})

	t.Run("No results", func(t *testing.T) {
		test := setupIngest(t, false)

		_, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		assert.ErrorIs(t, err, sighting.ErrIngestNoResults)
	})

	t.Run("GBIF error", func(t *testing.T) {
		test := setupIngest(t, false)
		test.gbif.err = errInternal

		_, err := test.svc.Ingest(test.ctx, sighting.IngestRequest{Limit: 10})
		assert.ErrorIs(t, err, errInternal)
	})
}

// paginationOf returns the page each search asked for.
func paginationOf(searches []gbif.OccurrencesSearchArgs) []gbif.PaginationOptions {
	out := make([]gbif.PaginationOptions, len(searches))
	for i, s := range searches {
		out[i] = s.PaginationOptions
	}
	return out
}
