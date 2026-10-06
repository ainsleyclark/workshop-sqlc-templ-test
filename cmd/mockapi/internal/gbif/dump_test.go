package gbif

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"workshop/internal/domain/species"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dumpCounts counts the rows the embedded dump inserts into each table.
func dumpCounts(t *testing.T) map[string]int {
	t.Helper()

	r, err := gzip.NewReader(bytes.NewReader(dump))
	require.NoError(t, err)

	counts := make(map[string]int)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for scanner.Scan() {
		rest, ok := strings.CutPrefix(scanner.Text(), "INSERT INTO ")
		if !ok {
			continue
		}
		table, values, _ := strings.Cut(rest, " VALUES ")
		if table == "occurrence_taxa" {
			counts[table] += strings.Count(values, "), (") + 1
		} else {
			counts[table]++
		}
	}
	require.NoError(t, scanner.Err())
	return counts
}

// TestEmbeddedDump builds the embedded dump into a SQLite file, the way the
// mock does on its first start, and checks that file directly: that every
// row made it in, that each is something ingest keeps, that every animal
// and a spread of years and places are there, and that searches come back
// in a random-looking order.
func TestEmbeddedDump(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "gbif.db")
	st, err := open(t.Context(), dump, path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	count := func(t *testing.T, query string, args ...any) int {
		t.Helper()
		var n int
		require.NoError(t, db.QueryRowContext(t.Context(), query, args...).Scan(&n), query)
		return n
	}

	t.Run("Every row is loaded", func(t *testing.T) {
		t.Parallel()

		want := dumpCounts(t)
		for _, table := range []string{"occurrences", "occurrence_taxa", "species", "media"} {
			assert.Positive(t, want[table], table)
			assert.Equal(t, want[table], count(t, "SELECT count(*) FROM "+table), table)
		}
		assert.Equal(t, want["occurrences"], count(t, "SELECT count(DISTINCT key) FROM occurrences"), "no duplicate occurrences")
	})

	t.Run("Every occurrence is complete", func(t *testing.T) {
		t.Parallel()

		for name, query := range map[string]string{
			"without taxa": `SELECT count(*) FROM occurrences
				WHERE key NOT IN (SELECT occurrence_key FROM occurrence_taxa)`,
			"without a species": `SELECT count(*) FROM occurrences
				WHERE json_extract(raw, '$.speciesKey') IS NOT NULL
				AND json_extract(raw, '$.speciesKey') NOT IN (SELECT key FROM species)`,
			"without species media": `SELECT count(*) FROM occurrences
				WHERE json_extract(raw, '$.speciesKey') IS NOT NULL
				AND json_extract(raw, '$.speciesKey') NOT IN (SELECT key FROM media)`,
			"without coordinates": `SELECT count(*) FROM occurrences WHERE lat IS NULL OR lng IS NULL`,
			"not present":         `SELECT count(*) FROM occurrences WHERE json_extract(raw, '$.occurrenceStatus') != 'PRESENT'`,
			"without a photo": `SELECT count(*) FROM occurrences WHERE NOT EXISTS (
				SELECT 1 FROM json_each(raw, '$.media') WHERE json_extract(value, '$.type') = 'StillImage')`,
		} {
			assert.Zero(t, count(t, query), "occurrences %s", name)
		}
	})

	t.Run("Every animal is there", func(t *testing.T) {
		t.Parallel()

		var missing []species.Animal
		for _, a := range species.Animals() {
			keys, err := json.Marshal(a.TaxonKeys())
			require.NoError(t, err)
			n := count(t, `SELECT count(DISTINCT occurrence_key) FROM occurrence_taxa
				WHERE taxon_key IN (SELECT value FROM json_each(?))`, string(keys))
			if n == 0 {
				missing = append(missing, a)
			}
		}
		assert.Empty(t, missing, "animals with no occurrences")
	})

	t.Run("Spread over years and places", func(t *testing.T) {
		t.Parallel()

		assert.GreaterOrEqual(t, count(t, "SELECT count(DISTINCT substr(event_date, 1, 4)) FROM occurrences"), 6, "years")
		assert.GreaterOrEqual(t, count(t, "SELECT count(DISTINCT country) FROM occurrences"), 30, "countries")
		for _, country := range themeCountries {
			assert.Positive(t, count(t, "SELECT count(*) FROM occurrences WHERE country = ?", country), country)
		}
	})

	t.Run("Stored order is random", func(t *testing.T) {
		t.Parallel()

		rows, err := db.QueryContext(t.Context(),
			"SELECT coalesce(json_extract(raw, '$.speciesKey'), 0) FROM occurrences ORDER BY shuffle, key")
		require.NoError(t, err)
		var order []int
		for rows.Next() {
			var k int
			require.NoError(t, rows.Scan(&k))
			order = append(order, k)
		}
		require.NoError(t, rows.Err())

		firstPage := make(map[int]bool)
		for _, k := range order[:min(maxLimit, len(order))] {
			firstPage[k] = true
		}
		assert.GreaterOrEqual(t, len(firstPage), 40, "species in the first page")

		longest, run := 1, 1
		for i := 1; i < len(order); i++ {
			if order[i] == order[i-1] {
				run++
				longest = max(longest, run)
			} else {
				run = 1
			}
		}
		assert.LessOrEqual(t, longest, 6, "longest run of one species")
	})

	t.Run("Searching for every animal comes back mixed", func(t *testing.T) {
		t.Parallel()

		q := url.Values{"limit": {strconv.Itoa(maxLimit)}, "checklistKey": {backbone}}
		for _, a := range species.Animals() {
			for _, k := range a.TaxonKeys() {
				q.Add("taxonKey", strconv.Itoa(k))
			}
		}
		status, body := get(t, st.Handler(), "/gbif/v1/occurrence/search?"+q.Encode())
		require.Equal(t, http.StatusOK, status, string(body))

		var p struct {
			Results []struct {
				SpeciesKey int `json:"speciesKey"`
			} `json:"results"`
		}
		require.NoError(t, json.Unmarshal(body, &p))
		require.Len(t, p.Results, maxLimit)

		seen := make(map[int]bool)
		for _, r := range p.Results {
			seen[r.SpeciesKey] = true
		}
		assert.GreaterOrEqual(t, len(seen), 40, "species in one page")
	})
}
