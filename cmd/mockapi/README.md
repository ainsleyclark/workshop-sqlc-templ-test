# Mock API

A local stand-in for the APIs the workshop calls, and what the app uses by default: a full room
ingesting at once trips GBIF's rate limit, and the venue's wifi can't be relied on. GBIF is served
under `/gbif/v1`, and a generated Open-Meteo mock under `/weather/v1`.

The app's CLI starts it in the background the first time a command needs it (`sightings ingest`,
`db seed`), so there's nothing to run by hand. To serve it yourself:

```bash
make mock                    # from the starter (or an attendee folder), listens on :8082
go run ./cmd/mockapi         # the same, or add serve --port 8082 to pick the port
```

To use the real APIs, pass `--live`: `make run-live`, or `workshop --live sightings ingest -n 20`.

## GBIF

The mock serves the three endpoints the starter uses: `/occurrence/search`, `/species/{key}` and
`/species/{key}/media`. Results are GBIF's own JSON, stored untouched, so they decode exactly as
they would from the real API.

Search supports `taxonKey` (any rank, so `212` finds every bird), `country`,
`decimalLatitude`/`decimalLongitude`, `geo_distance`, `eventDate`, `limit` and `offset`, following
GBIF's paging rules. One difference from GBIF: `count` is the number of matches in the mock's
data, not GBIF's global total.

### Data

The data is embedded from `internal/gbif/data/gbif.sql.gz`. To refresh it from the live API, run:

```bash
make mock-scrape             # or go run ./cmd/mockapi scrape
```

This searches every animal the starter knows (`internal/domain/species`), since ingest only ever
asks for those: once per year band, so the records span a decade rather than GBIF's newest few
weeks, and once in each theme country (Kenya and Antarctica). Every search asks only for what
ingest keeps: present, with a photo and clean coordinates. Then it fetches each species found and
its media. Pages are spaced out and `Retry-After` is honoured, so the scrape stays clear of GBIF's
rate limit; it takes a while.

The dump is in scrape order, a search at a time, so the mock orders results by a scrambled key
rather than as stored: a search for every animal at once comes back mixed rather than alphabetical,
and since the order is the same on every request, paging with an offset still sees each record
once. The seed list lives in `internal/gbif/scrape.go`; tune `perBand` and `perCountry` there to
keep the file around 50MB, well under GitHub's 100MB limit.

The first start on a machine unpacks the dump into a SQLite file in the user cache folder
(`workshop-mockapi/gbif-<hash>.db`), and later starts open it straight away. A new dump gets a new
file, and the old one is removed.

## Open-Meteo

The mock serves the one endpoint the starter uses: `/archive`, taking `latitude`, `longitude`,
`start_hour`, `end_hour` and `hourly`. Unlike GBIF's fixed set of occurrences, a sighting's
coordinates and event date can be almost anything, so there's no fixed dataset to record ahead of
time. Instead, each reading is generated deterministically from the request: the same place and
hour always return the same weather, so demos and tests stay reproducible, but nothing is scraped
or embedded.

## Layout

| Path | Purpose |
| --- | --- |
| `main.go` | Calls `cmd.Run()`. |
| `internal/common` (the app's) | The HTTP server and request logger are the same ones `workshop web` uses. |
| `internal/cmd` | The `serve` (default) and `scrape` commands, built with urfave/cli like the app's. |
| `internal/gbif` | The GBIF handlers, search filters, embedded data and scraper. |
| `internal/weather` | The Open-Meteo archive handler and the deterministic weather generator. |

Each mocked API lives in its own package, mounted at its own prefix in `internal/cmd/serve.go`.
