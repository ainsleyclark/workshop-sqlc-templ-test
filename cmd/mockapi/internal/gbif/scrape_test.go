package gbif

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// page is a GBIF search response holding one occurrence per key.
func page(keys ...int) []byte {
	results := make([]map[string]any, len(keys))
	for i, k := range keys {
		results[i] = map[string]any{"key": k, "countryCode": "KE", "eventDate": "2020-01-01", "speciesKey": 1}
	}
	b, _ := json.Marshal(map[string]any{"results": results})
	return b
}

// occurrenceKeys lists the occurrence keys in a gzipped dump, in order.
func occurrenceKeys(t *testing.T, gz []byte) []int {
	t.Helper()

	r, err := gzip.NewReader(bytes.NewReader(gz))
	require.NoError(t, err)
	sql, err := io.ReadAll(r)
	require.NoError(t, err)

	var keys []int
	for _, m := range regexp.MustCompile(`INSERT INTO occurrences VALUES \((\d+),`).FindAllSubmatch(sql, -1) {
		k, err := strconv.Atoi(string(m[1]))
		require.NoError(t, err)
		keys = append(keys, k)
	}
	return keys
}

func TestSeeds(t *testing.T) {
	t.Parallel()

	all := seeds()
	require.NotEmpty(t, all)

	for _, s := range all {
		assert.NotEmpty(t, s.query["taxonKey"], s.name)
		assert.Positive(t, s.limit, s.name)
		for k, v := range kept {
			assert.Equal(t, v, s.query[k], "%s: %s", s.name, k)
		}
		assert.True(t, s.query.Has("year") != s.query.Has("country"), "%s: a year band or a country", s.name)
	}
}

func TestScrape(t *testing.T) {
	t.Parallel()

	var limited atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/occurrence/search" && !limited.Swap(true):
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
		case r.URL.Path == "/occurrence/search":
			taxon, _ := strconv.Atoi(r.URL.Query().Get("taxonKey"))
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			var keys []int
			for i := offset; i < min(offset+limit, 3); i++ {
				keys = append(keys, taxon*10+i)
			}
			_, _ = w.Write(page(keys...))
		case r.URL.Path == "/species/1":
			_, _ = fmt.Fprint(w, `{"key":1}`)
		case r.URL.Path == "/species/1/media":
			_, _ = fmt.Fprint(w, `{"results":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(upstream.Close)

	newSeed := func(taxon, limit int) seed {
		return seed{name: strconv.Itoa(taxon), query: map[string][]string{"taxonKey": {strconv.Itoa(taxon)}}, limit: limit}
	}
	out := filepath.Join(t.TempDir(), "gbif.sql.gz")
	err := scrape(t.Context(), slog.New(slog.DiscardHandler), scrapeConfig{
		BaseURL: upstream.URL,
		Seeds:   []seed{newSeed(1, 10), newSeed(2, 2)},
		Out:     out,
	})
	require.NoError(t, err)

	gz, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, []int{10, 11, 12, 20, 21}, occurrenceKeys(t, gz), "in scrape order, with seed 2 capped at its limit")
	assert.True(t, limited.Load(), "the rate-limited request was retried")
}
