// Package gbif mocks the parts of the GBIF API the workshop uses, serving
// GBIF's own responses from an embedded SQLite dump.
package gbif

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"modernc.org/sqlite"
)

// dump is the scraped GBIF data, one SQL statement per line.
// Regenerate it with `go run ./cmd/mockapi scrape`.
//
//go:embed data/gbif.sql.gz
var dump []byte

// schema keeps GBIF's JSON untouched in raw columns, so responses are
// exactly what GBIF sent, and copies out only the fields we filter on.
const schema = `
-- shuffle orders search results; see shuffleKey. It's generated, so the
-- dump's INSERTs don't supply it.
CREATE TABLE occurrences (
	key        INTEGER NOT NULL UNIQUE,
	country    TEXT NOT NULL,
	lat        REAL,
	lng        REAL,
	event_date TEXT NOT NULL,
	raw        TEXT NOT NULL,
	shuffle    INTEGER GENERATED ALWAYS AS (shuffle_key(key)) STORED
);
-- Every rank an occurrence belongs to, so taxonKey=212 finds all birds.
CREATE TABLE occurrence_taxa (
	occurrence_key INTEGER NOT NULL,
	taxon_key      INTEGER NOT NULL
);
CREATE TABLE species (key INTEGER PRIMARY KEY, raw TEXT NOT NULL);
CREATE TABLE media   (key INTEGER PRIMARY KEY, raw TEXT NOT NULL);
`

const indexes = `
CREATE INDEX occurrences_shuffle ON occurrences (shuffle, key);
CREATE INDEX occurrence_taxa_taxon ON occurrence_taxa (taxon_key, occurrence_key);
CREATE INDEX occurrences_country ON occurrences (country);
CREATE INDEX occurrences_lat_lng ON occurrences (lat, lng);
`

func init() {
	sqlite.MustRegisterDeterministicScalarFunction("shuffle_key", 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			key, _ := args[0].(int64)
			return shuffleKey(key), nil
		})

	// distance_km lets geo_distance searches filter and paginate in SQL.
	sqlite.MustRegisterDeterministicScalarFunction("distance_km", 4,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			var f [4]float64
			for i, v := range args {
				n, ok := v.(float64)
				if !ok {
					return nil, nil // NULL coordinates never match.
				}
				f[i] = n
			}
			return haversineKm(f[0], f[1], f[2], f[3]), nil
		})
}

// shuffleKey scrambles an occurrence key (splitmix64's finaliser), so
// ordering by it looks random: neighbours share no animal, year or place.
// The dump is in scrape order, a seed search at a time, so without it a
// search for every animal would return all the African buffalo first.
// Unlike ORDER BY random() it's the same on every request, so paging with
// an offset still sees every record exactly once.
func shuffleKey(key int64) int64 {
	z := uint64(key) + 0x9e3779b97f4a7c15
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return int64((z ^ (z >> 31)) >> 1) // Kept positive, so it sorts as written.
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0088
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

// Store serves GBIF data from a SQLite database.
type Store struct {
	db *sql.DB
}

// Open loads the embedded GBIF data. The dump is far too big to hold in
// memory, so the first start on a machine builds it into a SQLite file in
// the user's cache, which every later start opens straight away. Without a
// cache folder it falls back to memory.
func Open(ctx context.Context) (*Store, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return open(ctx, dump, "")
	}
	dir = filepath.Join(dir, "workshop-mockapi")
	if err = os.MkdirAll(dir, 0o750); err != nil {
		return open(ctx, dump, "")
	}

	// Named for the dump, so a new scrape gets a fresh build.
	sum := sha256.Sum256(dump)
	name := "gbif-" + hex.EncodeToString(sum[:6]) + ".db"
	removeStale(dir, name)

	return open(ctx, dump, filepath.Join(dir, name))
}

// removeStale deletes databases built from older dumps.
func removeStale(dir, keep string) {
	old, _ := filepath.Glob(filepath.Join(dir, "gbif-*.db"))
	for _, path := range old {
		if filepath.Base(path) != keep {
			_ = os.Remove(path)
		}
	}
}

// open serves a gzipped dump from the database at path, building it first
// if it isn't there. An empty path loads it into memory instead.
func open(ctx context.Context, gz []byte, path string) (*Store, error) {
	if path == "" {
		db, err := sql.Open("sqlite", "file::memory:")
		if err != nil {
			return nil, err
		}
		// Each connection to :memory: is its own empty database, so keep one.
		db.SetMaxOpenConns(1)
		if err = load(ctx, db, gz); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("loading GBIF data: %w", err)
		}
		return &Store{db: db}, nil
	}

	if _, err := os.Stat(path); err != nil {
		if err = build(ctx, gz, path); err != nil {
			return nil, fmt.Errorf("loading GBIF data: %w", err)
		}
	}

	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// build loads the dump into a new database file. It writes to a file of its
// own and renames it into place when done, so a second mock starting at the
// same time, or one stopped halfway, never leaves a partial database behind.
func build(ctx context.Context, gz []byte, path string) error {
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	defer os.Remove(tmp) //nolint:errcheck // Gone already once renamed.

	// Nothing reads the file until it's renamed, so skip the journal.
	db, err := sql.Open("sqlite", "file:"+tmp+"?_pragma=journal_mode(OFF)&_pragma=synchronous(OFF)")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	if err = load(ctx, db, gz); err != nil {
		_ = db.Close()
		return err
	}
	if err = db.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func load(ctx context.Context, db *sql.DB, gz []byte) error {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err = tx.ExecContext(ctx, schema); err != nil {
		return err
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		if _, err = tx.ExecContext(ctx, scanner.Text()); err != nil {
			return err
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, indexes); err != nil {
		return err
	}
	return tx.Commit()
}

// Stats counts the occurrences and species loaded.
func (s *Store) Stats(ctx context.Context) (occurrences, species int, err error) {
	err = s.db.QueryRowContext(ctx,
		"SELECT (SELECT count(*) FROM occurrences), (SELECT count(*) FROM species)",
	).Scan(&occurrences, &species)
	return occurrences, species, err
}

// Close releases the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// dataset is everything a scrape collected, ready to be written as SQL.
type dataset struct {
	Occurrences []occurrence
	Species     map[int]json.RawMessage
	Media       map[int]json.RawMessage
	seen        map[int64]bool
}

func newDataset() *dataset {
	return &dataset{
		Species: make(map[int]json.RawMessage),
		Media:   make(map[int]json.RawMessage),
		seen:    make(map[int64]bool),
	}
}

// occurrence is a GBIF occurrence as it came over the wire,
// plus the handful of fields the search endpoint filters on.
type occurrence struct {
	Key       int64
	Country   string
	Lat, Lng  *float64
	EventDate string // The first day the eventDate covers, as YYYY-MM-DD.
	TaxonKeys []int
	Raw       json.RawMessage
}

// addPage adds every new occurrence in a GBIF search response, keeping the
// order GBIF returned them in. It reports how many results the page held.
func (d *dataset) addPage(body []byte) (int, error) {
	var page struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return 0, err
	}
	for _, raw := range page.Results {
		o, err := parseOccurrence(raw)
		if err != nil {
			return 0, err
		}
		if d.seen[o.Key] {
			continue
		}
		d.seen[o.Key] = true
		d.Occurrences = append(d.Occurrences, o)
	}
	return len(page.Results), nil
}

// speciesKeys lists every species the occurrences belong to.
func (d *dataset) speciesKeys() []int {
	keys := make(map[int]bool)
	for _, o := range d.Occurrences {
		var v struct {
			SpeciesKey int `json:"speciesKey"`
		}
		if json.Unmarshal(o.Raw, &v) == nil && v.SpeciesKey != 0 {
			keys[v.SpeciesKey] = true
		}
	}
	return slices.Sorted(maps.Keys(keys))
}

func parseOccurrence(raw json.RawMessage) (occurrence, error) {
	var v struct {
		Key              int64    `json:"key"`
		CountryCode      string   `json:"countryCode"`
		DecimalLatitude  *float64 `json:"decimalLatitude"`
		DecimalLongitude *float64 `json:"decimalLongitude"`
		EventDate        string   `json:"eventDate"`
		TaxonKey         int      `json:"taxonKey"`
		KingdomKey       int      `json:"kingdomKey"`
		PhylumKey        int      `json:"phylumKey"`
		ClassKey         int      `json:"classKey"`
		OrderKey         int      `json:"orderKey"`
		FamilyKey        int      `json:"familyKey"`
		GenusKey         int      `json:"genusKey"`
		SpeciesKey       int      `json:"speciesKey"`
		AcceptedTaxonKey int      `json:"acceptedTaxonKey"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return occurrence{}, fmt.Errorf("parsing occurrence: %w", err)
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return occurrence{}, err
	}

	var taxa []int
	for _, k := range []int{
		v.TaxonKey, v.KingdomKey, v.PhylumKey, v.ClassKey, v.OrderKey,
		v.FamilyKey, v.GenusKey, v.SpeciesKey, v.AcceptedTaxonKey,
	} {
		if k != 0 && !slices.Contains(taxa, k) {
			taxa = append(taxa, k)
		}
	}

	return occurrence{
		Key:       v.Key,
		Country:   v.CountryCode,
		Lat:       v.DecimalLatitude,
		Lng:       v.DecimalLongitude,
		EventDate: firstDay(v.EventDate),
		TaxonKeys: taxa,
		Raw:       compact.Bytes(),
	}, nil
}

// firstDay turns a GBIF eventDate, which may be partial ("2020-01") or an
// interval ("2020-01-01/2020-02-01"), into the first day it covers.
func firstDay(eventDate string) string {
	start, _, _ := strings.Cut(eventDate, "/")
	start = start[:min(len(start), 10)]
	switch len(start) {
	case 4:
		return start + "-01-01"
	case 7:
		return start + "-01"
	}
	return start
}

// writeSQL writes the dataset as gzipped INSERTs, one per line. The output
// only depends on the data, so re-scraping unchanged data is a no-op diff.
func (d *dataset) writeSQL(w io.Writer) error {
	gz := gzip.NewWriter(w)
	bw := bufio.NewWriter(gz)

	for _, o := range d.Occurrences {
		fmt.Fprintf(bw, "INSERT INTO occurrences VALUES (%d, %s, %s, %s, %s, %s);\n",
			o.Key, quote(o.Country), float(o.Lat), float(o.Lng), quote(o.EventDate), quote(string(o.Raw)))
		if len(o.TaxonKeys) == 0 {
			continue
		}
		values := make([]string, len(o.TaxonKeys))
		for i, k := range o.TaxonKeys {
			values[i] = fmt.Sprintf("(%d, %d)", o.Key, k)
		}
		fmt.Fprintf(bw, "INSERT INTO occurrence_taxa VALUES %s;\n", strings.Join(values, ", "))
	}
	for _, table := range []struct {
		name string
		rows map[int]json.RawMessage
	}{{"species", d.Species}, {"media", d.Media}} {
		for _, key := range slices.Sorted(maps.Keys(table.rows)) {
			var compact bytes.Buffer
			if err := json.Compact(&compact, table.rows[key]); err != nil {
				return fmt.Errorf("%s %d: %w", table.name, key, err)
			}
			fmt.Fprintf(bw, "INSERT INTO %s VALUES (%d, %s);\n", table.name, key, quote(compact.String()))
		}
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	return gz.Close()
}

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func float(f *float64) string {
	if f == nil {
		return "NULL"
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}
