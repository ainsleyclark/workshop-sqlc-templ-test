# Full-Stack Go: Domain-Driven Applications with sqlc and templ

**Test Workshop**

The workshop app: a site for **animal sightings**. Occurrence records come from
[GBIF](https://www.gbif.org), are stored in SQLite through [sqlc](https://sqlc.dev), rendered
server-side with [templ](https://templ.guide), and enriched with
[Open-Meteo](https://open-meteo.com) weather.

This repository is yours for the day. It's the finished app: we walk through it together, then
you make the site your own in your fork and push it there.

## Before you start

You need Go (the version in `go.mod`), Git, `make` and a GitHub account. For the Theming block
you'll also want [Claude Code](https://code.claude.com/docs/en/setup) and Python 3. GoLand is
the preferred IDE for this workshop, and you should have received a promo code to use it.

|             | macOS                                        | Linux                                   | Windows                                  |
|-------------|----------------------------------------------|-----------------------------------------|------------------------------------------|
| Go          | `brew install go`                            | `sudo snap install go --classic`        | the `.msi` from https://go.dev/dl        |
| Git         | comes with `make` below                      | `sudo apt install git`                  | `winget install Git.Git`                 |
| `make`      | `xcode-select --install`                     | `sudo apt install make`                 | `winget install ezwinports.make`         |
| Claude Code | `curl -fsSL https://claude.ai/install.sh \| bash` | the same                           | `irm https://claude.ai/install.ps1 \| iex` |
| Python 3    | `brew install python`                        | `sudo apt install python3`              | `winget install Python.Python.3.12`      |

You need Go 1.27.1 or newer: `go version` should say so, as setup won't download a newer one
for you. (winget's Go lags behind Go's releases, hence the installer on Windows.)

On Windows, open a new terminal after installing so the new commands are on your PATH, and run
everything from PowerShell or Git Bash. (`apt` is Debian and Ubuntu; use your distribution's
package manager otherwise.)

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
   downloaded.

   ```sh
   make setup
   ```

   It also installs the app as `workshop` in `~/go/bin`, so the commands the CLI's hints name,
   like `workshop sightings find <id>`, work from anywhere. `make run` and `make web` always run
   the code as it is now; `workshop` is as new as your last `make setup`.

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

**On Windows**, run these from PowerShell or Git Bash. Every recipe in the `Makefile` is a plain
`go` or `git` command, so if `make` won't cooperate, open `tools/setup.mk` and run the lines under
`tools:` and `setup:` yourself. If `vendor/` ever fails you, `make tools-online` installs the
same tools at the same versions from the internet. Only `make help` and `make kill-ports` need a
Unix shell.

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
