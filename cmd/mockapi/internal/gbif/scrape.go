package gbif

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	"workshop/internal/clients/gbif"
	"workshop/internal/domain/species"
)

// perBand and perCountry cap the occurrences fetched per seed search, for
// an animal's year band and for an animal in one country. Tune them to keep
// the file around 50MB, well under GitHub's 100MB limit.
const (
	perBand    = 400
	perCountry = 300
)

// pageDelay spaces out occurrence pages, to stay clear of GBIF's rate limit.
const pageDelay = 250 * time.Millisecond

type scrapeConfig struct {
	BaseURL   string        // The GBIF API to copy.
	Seeds     []seed        // The searches to copy.
	PageDelay time.Duration // Pause between occurrence pages.
	Out       string        // Where to write the gzipped SQL.
}

// dataPath finds the embedded data file from the starter, the mockapi
// folder or the repository root, whichever the scrape is run from.
func dataPath() string {
	const path = "internal/gbif/data/gbif.sql.gz"
	for _, dir := range []string{".", "cmd/mockapi", "starter/cmd/mockapi"} {
		p := filepath.Join(dir, path)
		if _, err := os.Stat(filepath.Dir(p)); err == nil {
			return p
		}
	}
	return path
}

// backbone is the GBIF Backbone checklist, which the taxon keys belong to.
const backbone = "d7dddbf4-2cf0-4f39-9b2a-bb099caae36c"

// seed is one occurrence search whose results go into the mock.
type seed struct {
	name  string
	query url.Values
	limit int // Maximum occurrences to fetch.
}

// yearBands spread each animal over a decade. GBIF returns the newest
// records first, so a single search would all land in the same few weeks.
var yearBands = []string{"2016,2019", "2020,2022", "2023,2024", "2025,2026"}

// themeCountries are searched for every animal on top of the year bands,
// so ingests with -c KE and the themes built around them find plenty.
var themeCountries = []string{"KE", "AQ"}

// kept limits every seed to what the starter's ingest keeps: records of an
// animal actually present, with a photo and a usable location. Anything
// else would be scraped, shipped and then skipped.
var kept = url.Values{
	"mediaType":          {"StillImage"},
	"occurrenceStatus":   {"PRESENT"},
	"hasCoordinate":      {"true"},
	"hasGeospatialIssue": {"false"},
}

// seeds covers every animal the app knows, since the starter only ever
// searches for those, in each year band and each theme country.
func seeds() []seed {
	var out []seed
	add := func(name string, a species.Animal, limit int, extra url.Values) {
		q := url.Values{"checklistKey": {backbone}}
		for _, k := range a.TaxonKeys() {
			q.Add("taxonKey", strconv.Itoa(k))
		}
		for k, v := range kept {
			q[k] = v
		}
		for k, v := range extra {
			q[k] = v
		}
		out = append(out, seed{name: name, query: q, limit: limit})
	}

	for _, a := range species.Animals() {
		for _, years := range yearBands {
			add(a.String()+" "+years, a, perBand, url.Values{"year": {years}})
		}
	}
	for _, country := range themeCountries {
		for _, a := range species.Animals() {
			add(a.String()+" in "+country, a, perCountry, url.Values{"country": {country}})
		}
	}

	return out
}

// Scrape copies the live GBIF API into the data file the mock embeds.
func Scrape(ctx context.Context, logger *slog.Logger) error {
	return scrape(ctx, logger, scrapeConfig{
		BaseURL:   gbif.BaseURL,
		Seeds:     seeds(),
		PageDelay: pageDelay,
		Out:       dataPath(),
	})
}

func scrape(ctx context.Context, logger *slog.Logger, cfg scrapeConfig) error {
	c := &upstream{baseURL: cfg.BaseURL, http: &http.Client{Timeout: time.Minute}}
	d := newDataset()

	for i, s := range cfg.Seeds {
		before := len(d.Occurrences)
		for offset := 0; offset < s.limit; offset += maxLimit {
			limit := min(maxLimit, s.limit-offset)
			q := url.Values{}
			for k, v := range s.query {
				q[k] = v
			}
			q.Set("limit", strconv.Itoa(limit))
			q.Set("offset", strconv.Itoa(offset))

			if err := sleep(ctx, cfg.PageDelay); err != nil {
				return err
			}
			body, err := c.get(ctx, "/occurrence/search", q)
			if err != nil {
				return fmt.Errorf("scraping %s: %w", s.name, err)
			}
			n, err := d.addPage(body)
			if err != nil {
				return fmt.Errorf("scraping %s: %w", s.name, err)
			}
			if n < limit {
				break // No more records.
			}
		}
		logger.Info("Scraped occurrences",
			"seed", s.name,
			"done", fmt.Sprintf("%d/%d", i+1, len(cfg.Seeds)),
			"new", len(d.Occurrences)-before,
			"total", len(d.Occurrences),
		)
	}

	keys := d.speciesKeys()
	logger.Info("Scraping species and media", "species", len(keys))
	if err := c.species(ctx, logger, d, keys); err != nil {
		return err
	}

	// Writes are scoped to the data folder, so nothing outside it
	// can be touched whatever the path says.
	root, err := os.OpenRoot(filepath.Dir(cfg.Out))
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck

	name := filepath.Base(cfg.Out)
	f, err := root.Create(name + ".tmp")
	if err != nil {
		return err
	}
	if err = d.writeSQL(f); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = root.Rename(name+".tmp", name); err != nil {
		return err
	}

	info, err := root.Stat(name)
	if err != nil {
		return err
	}
	logger.Info("Wrote GBIF data",
		"file", cfg.Out,
		"size", fmt.Sprintf("%.1fMB", float64(info.Size())/(1<<20)),
		"occurrences", len(d.Occurrences),
		"species", len(d.Species),
	)
	return nil
}

// species fetches each species and its media, a few at a time,
// as there are far more of them than occurrence pages.
func (c *upstream) species(ctx context.Context, logger *slog.Logger, d *dataset, keys []int) error {
	const workers = 4

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
		done     int
	)
	jobs := make(chan int)
	for range workers {
		wg.Go(func() {
			for key := range jobs {
				path := "/species/" + strconv.Itoa(key)
				sp, err := c.get(ctx, path, nil)
				var media []byte
				if err == nil {
					media, err = c.get(ctx, path+"/media", nil)
				}

				mu.Lock()
				switch {
				case errors.Is(err, errNotFound):
				case err != nil && firstErr == nil:
					firstErr = fmt.Errorf("species %d: %w", key, err)
				case err != nil:
				default:
					d.Species[key] = sp
					d.Media[key] = media
				}
				if done++; done%250 == 0 {
					logger.Info("Scraped species", "done", done, "of", len(keys))
				}
				mu.Unlock()
			}
		})
	}

	for _, key := range keys {
		mu.Lock()
		failed := firstErr != nil
		mu.Unlock()
		if failed || ctx.Err() != nil {
			break
		}
		jobs <- key
	}
	close(jobs)
	wg.Wait()

	return errors.Join(firstErr, ctx.Err())
}

// upstream fetches raw response bodies from the real API.
// It deliberately skips the typed gbif client, since decoding and
// re-encoding would lose fields and change how values are written.
type upstream struct {
	baseURL string
	http    *http.Client
}

var errNotFound = errors.New("not found")

// get retries rate limits and server errors with a growing wait, or as
// long as GBIF's Retry-After asks.
func (c *upstream) get(ctx context.Context, path string, q url.Values) ([]byte, error) {
	uri := c.baseURL + path
	if len(q) > 0 {
		uri += "?" + q.Encode()
	}

	var (
		lastErr error
		wait    time.Duration
	)
	for attempt := range 6 {
		if attempt > 0 {
			// A Retry-After is honoured, up to a couple of minutes.
			wait = min(wait, 2*time.Minute)
			if err := sleep(ctx, max(wait, time.Second<<(attempt-1))); err != nil {
				return nil, err
			}
		}

		body, status, retryAfter, err := c.do(ctx, uri)
		wait = retryAfter
		switch {
		case err != nil:
			lastErr = err
		case status == http.StatusNotFound:
			return nil, errNotFound
		case status == http.StatusTooManyRequests || status >= 500:
			lastErr = fmt.Errorf("GET %s: %s", uri, http.StatusText(status))
		case status != http.StatusOK:
			return nil, fmt.Errorf("GET %s: %s", uri, http.StatusText(status))
		default:
			return body, nil
		}
	}
	return nil, lastErr
}

// do makes one request, reporting how long a Retry-After header in seconds
// asks to wait, if there is one.
func (c *upstream) do(ctx context.Context, uri string) ([]byte, int, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, 0, err
	}
	defer res.Body.Close() //nolint:errcheck

	var retryAfter time.Duration
	if secs, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && secs > 0 {
		retryAfter = time.Duration(secs) * time.Second
	}

	body, err := io.ReadAll(res.Body)
	return body, res.StatusCode, retryAfter, err
}

// sleep waits for d, or until ctx is done.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
