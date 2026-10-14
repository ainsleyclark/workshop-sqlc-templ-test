package site

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"
	"workshop/internal/domain/sighting"
	"workshop/internal/domain/sighting/country"
	"workshop/internal/domain/species"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lister records the filter it was asked for and hands back whatever
// the test gave it.
type lister struct {
	sightings []sighting.Sighting
	err       error
	got       sighting.ListFilter
}

func (l *lister) List(_ context.Context, filter sighting.ListFilter) ([]sighting.Sighting, error) {
	l.got = filter
	return l.sightings, l.err
}

func TestHandler_Routes(t *testing.T) {
	t.Parallel()

	humpback := sighting.Sighting{
		ID:         uuid.New(),
		HappenedAt: time.Date(2025, 8, 14, 9, 30, 0, 0, time.UTC),
		RecordedBy: "Watamu Marine Association",
		Location: sighting.Location{
			Locality:    "Watamu",
			Country:     "Kenya",
			CountryCode: country.Kenya,
			Coordinates: sighting.Coordinates{Latitude: -3.35, Longitude: 40.02},
		},
		Species: species.Species{
			VernacularName: "Humpback whale",
			ScientificName: "Megaptera novaeangliae",
		},
	}

	serve := func(t *testing.T, l Lister, path string) *httptest.ResponseRecorder {
		t.Helper()

		mux := http.NewServeMux()
		New(slog.New(slog.DiscardHandler), l).Routes(mux)
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil).WithContext(t.Context())
		mux.ServeHTTP(response, request)

		return response
	}

	t.Run("Renders each theme with its filter", func(t *testing.T) {
		t.Parallel()

		tt := map[string]struct {
			path  string
			title string
			want  sighting.ListFilter
		}{
			"Antarctica": {
				path:  "/antarctica",
				title: "Antarctic Station",
				want:  sighting.ListFilter{Country: country.Antarctica},
			},
			"Kenya": {
				path:  "/kenya",
				title: "Kenyan Coast",
				want:  sighting.ListFilter{Country: country.Kenya},
			},
			"Humpback": {
				path:  "/humpback",
				title: "Humpback Console",
				want:  sighting.ListFilter{Animals: []species.Animal{species.AnimalHumpbackWhale}},
			},
		}

		for name, test := range tt {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				l := &lister{sightings: []sighting.Sighting{humpback}}
				response := serve(t, l, test.path)

				assert.Equal(t, http.StatusOK, response.Code)
				assert.Equal(t, "text/html; charset=utf-8", response.Header().Get("Content-Type"))
				assert.Contains(t, response.Body.String(), test.title)
				assert.Contains(t, response.Body.String(), "Humpback whale")
				assert.Equal(t, test.want, l.got)
			})
		}
	})

	t.Run("Serves the assets the themes load", func(t *testing.T) {
		t.Parallel()

		for _, path := range []string{
			"/assets/css/antarctica.css",
			"/assets/css/kenya.css",
			"/assets/css/humpback.css",
			"/assets/js/sightings-map.js",
		} {
			response := serve(t, &lister{}, path)
			assert.Equal(t, http.StatusOK, response.Code, path)
		}
	})

	t.Run("Redirects the front door to Kenya", func(t *testing.T) {
		t.Parallel()

		response := serve(t, &lister{}, "/")

		assert.Equal(t, http.StatusFound, response.Code)
		assert.Equal(t, "/kenya", response.Header().Get("Location"))
	})

	t.Run("Unknown paths are not found", func(t *testing.T) {
		t.Parallel()

		response := serve(t, &lister{}, "/sightings")

		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("Listing error", func(t *testing.T) {
		t.Parallel()

		response := serve(t, &lister{err: errors.New("database is locked")}, "/kenya")

		require.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Contains(t, response.Body.String(), "Unable to load sightings.")
		assert.NotContains(t, response.Body.String(), "database is locked")
	})
}
