# Themes

The three themed pages shown at the start of the workshop: finished examples of what the day builds
towards, rendered from the same `workshop.db` the starter reads. They're for showing, not for
building on.

```bash
make themes                  # from starter/, listens on :8083
go run ./cmd/themes          # the same from starter/; add --port or --db to change either
```

| Page          | Filter             | What it shows                                                        |
|---------------|--------------------|----------------------------------------------------------------------|
| `/antarctica` | country `AQ`       | A polar station: a map-led instrument panel and a station log        |
| `/kenya`      | country `KE`       | Field notes: an editorial issue, each record written up as a piece   |
| `/humpback`   | the humpback whale | A console: dense panels, a picture wall and a ledger of every record |

`/` redirects to `/kenya`.

## Why it's separate

Each theme is the same data through a different lens: a filter saying what the page is about, and a
package that presents what comes back in its own way. That makes them a good demonstration of a
bounded context, and a poor thing to build on, since each needs far more from the layout than a
first page does.

So they live here rather than in `web/views/`, and the starter knows nothing about them:

- **Nothing in the starter imports them.** Everything sits under `internal/`, so the compiler
  refuses any import from outside `cmd/themes`.
- **They have their own layout** (`internal/layout`), a copy of the starter's grown for fonts,
  skip links and the Leaflet map, and their own navigation.
- **They serve their own assets** (`internal/site/assets`), so the starter's `web/assets` holds
  only what its pages use.
- **They read sightings through a one-method interface** (`site.Lister`), satisfied by the
  sighting store, so they don't depend on the starter's handlers or service.
- **Attendees get them as examples.** `publish` copies this folder into the attendee repository,
  so `make themes` works in a fork, but the app they theme is still the one basic page in `web/`.

The one thing they share is the sighting domain. Change `sighting.Sighting` or `ListFilter` in the
starter and these may need updating to match.

## Layout

```
cmd/themes/
├── main.go             reads config, opens the database, serves the site
└── internal/
    ├── site/           the routes, the list → render plumbing, and the embedded assets
    ├── layout/         the document shell and the navigation between themes
    ├── themeutil/      formatting shared by the three themes
    ├── antarctica/
    ├── humpback/
    └── kenya/          each is a Props builder and a .templ file
```

Run `make templ` from `starter/` after editing a `.templ` file here, as anywhere else.
