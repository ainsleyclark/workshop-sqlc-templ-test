# Full-Stack Go: Domain-Driven Applications with sqlc and templ

**Test Workshop**

The workshop app: a site for **animal sightings**. Occurrence records come from
[GBIF](https://www.gbif.org), are stored in SQLite through [sqlc](https://sqlc.dev), rendered
server-side with [templ](https://templ.guide), and enriched with
[Open-Meteo](https://open-meteo.com) weather.

This repository is yours for the day. It's the finished app: we walk through it together, then
you make the site your own in your fork and push it there.

## Before you start

You need Go (the version in `go.mod`), Git and a GitHub account. GoLand is the preferred IDE for 
this workshop, and you should have received a promo code to use it.

1. **Fork this repository** on GitHub: the *Fork* button at the top
   of https://github.com/ainsleyclark/workshop-sqlc-templ-test. If you have the GitHub CLI,
   `gh repo fork ainsleyclark/workshop-sqlc-templ-test --clone` forks and clones in one go.
2. **Clone your fork** and open it:

   ```sh
   git clone git@github.com:<your-github-user>/workshop-sqlc-templ-test.git
   cd workshop-sqlc-templ-test
   ```

   Open this folder in the IDE. The `go.mod` is at the root, so there is nothing else to point 
   it at.
3. **Run setup once.** It builds the Go tools the day needs from `vendor/`, so nothing is 
   downloaded, offers to install Claude Code for the Theming block.

   ```sh
   make setup
   ```

## During the day

```sh
make run        # the interactive CLI: migrate, seed, ingest sightings, list, stats, web
make web        # serve the site on :8080
make web-hot    # the same, with templ hot reload
make templ      # regenerate templ after editing a .templ file
make sqlc       # regenerate sqlc after editing a query or migration
make test       # the tests
make update     # pull the instructor's latest changes
make push       # commit everything and push it to your fork
make help       # everything else
```

Your copy starts without a database: `go run . db seed` fills `workshop.db` with sightings, then
`make web` shows the site.
`make run`, then `sightings` → `ingest`, fetches more. Ingest reads from a mock of GBIF and
Open-Meteo bundled with the app, so it works offline and never hits GBIF's rate limit; it starts
by itself, and `make run-live` uses the real APIs instead. If the database gets into a
mess, `git checkout workshop.db` puts back the copy you last committed.

Commit and push to **your fork** as you go (`make push`), `workshop.db` included, so the instructor
can see what you've built. `/design-site` does this for you after each of Claude's turns.

## Where things are

```
main.go                     thin; hands off to internal/cmd
migrations/                 goose migrations
queries/                    sqlc queries, a file per table
internal/
├── cmd/                    CLI commands and the interactive menu
├── domain/                 the rules: sighting, species, weather — and their stores
├── clients/                outbound adapters: gbif, openmeteo
├── infra/                  config, sqlite, sqlc output
└── common/                 small generic helpers
web/                        your view layer: web.go owns the routes, handlers/, views/, assets/
architectures/              the opening tour, a ladder of five layouts; read, not typed in
.claude/, .junie/           Claude Code and Junie: settings and the /design-site skill for the Theming block
```

Each of `internal/{domain,clients,infra,common}` has a README saying where its line falls.
`SYLLABUS.md` is the course outline you were sent beforehand.
