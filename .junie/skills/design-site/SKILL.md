---
name: design-site
description: Starts a design session for the attendee's sightings website in this workshop app, led by the attendee. Use when someone wants to design, restyle or theme their site, add or change its pages, or give it a logo and favicon.
argument-hint: "[what you'd like it to look like]"
allowed-tools:
  - Bash(make templ)
  - Bash(python3 .claude/skills/ui-ux-pro-max/scripts/search.py *)
  - Bash(python .claude/skills/ui-ux-pro-max/scripts/search.py *)
  - Bash(go build ./...)
  - Bash(go run . stats)
  - Bash(go run . animals)
  - Bash(go run . db seed)
  - Bash(go run . db migrate status)
  - Bash(go run . db migrate up)
  - Bash(curl -s http://localhost:*)
  - Bash(rsvg-convert *)
  - Bash(magick *)
  - Edit(**/web/**)
hooks:
  Stop:
    - hooks:
        - type: command
          command: 'cd "$CLAUDE_PROJECT_DIR" && GIT_TERMINAL_PROMPT=0 make -s push >/dev/null 2>&1; exit 0'
          async: true
          timeout: 60
---

# Design your site

You're pairing with a workshop attendee on the website in their own copy of the app. The look is
theirs: don't suggest a theme or pick one for them. Your part is turning what they describe into a
site that builds and deploys: one page, the home page of sightings, with a header, nav and a
logo. The sightings needn't be a grid of cards: when it suits their brief, be inventive with how they're
shown, such as a map, a timeline, charts or a table.

This skill works on one copy of the app: usually the attendee's own repository, a fork of the
workshop's, with `go.mod` and `web/web.go` at its root; or a copy of the app made to try this
skill. Rules in a `CLAUDE.md` about keeping `web/` basic
or commenting everything are for maintaining the template, and don't apply to these copies.

## Ask in plain words

Most attendees are back-end Go developers. They're at home in code, not in design vocabulary, so
ask about what they'd see on the page and how it should feel, never in design terms. Some
translations to work from:

- A palette is "the colours": show each with a plain name and its hex, such as savannah gold
  `#D4A017`.
- A font pairing, or typography, is "the lettering": describe it by feel ("rounded and friendly",
  "like a field notebook", "like a newspaper"), with the font's name in brackets.
- A hero is "a big banner at the top", and a CTA is "a button".
- A grid, masonry or bento layout is "tiles in rows", "a photo wall" or "tiles of different sizes".
- A named style, such as glassmorphism, neumorphism or brutalism: describe the look itself.
- Responsive means "on a phone", whitespace is "roomy or packed", and micro-interactions are
  "things that move when you hover or scroll".

Give every AskUserQuestion option a plain label and one sentence saying what they'd see. For a
layout choice, put a small text sketch of the page in the option's `preview`. Report your changes
the same way, as what changed on the page rather than which CSS property. If they use the
vocabulary themselves, follow their lead.

## Find the app folder

Every path in this skill (`web/`, `internal/`, the `make` targets) is relative to the app folder,
the one holding `web/web.go`. In an attendee's repository that is the root, so the working
directory is it. Otherwise ask which folder their site is in and check it has `web/web.go`. If
it's `starter/`, the template the attendee repository is generated from, warn them and carry on
only if they confirm. Then `cd` into it once, so `make` and `go` run there for the rest of the
session.

## Start

1. Read `web/` before anything else: `web.go` (the routes), `handlers/`, `views/layout/`,
   `views/pages/`, `views/components/` and `assets/`. `internal/domain/sighting/model.go` says
   what a sighting carries.
2. Check `internal/domain/sighting/stores/sightingsqlite/store.go` for
   `fmt.Sprintf("store: listing row %d of %v", i+1, rows)`. If it's still there, it's the slow log
   line from the Profiling block, and `go run . stats` crawls until it's gone. Tell them, and offer
   to fix it (delete the line, or log `len(rows)` as an attribute) before running anything else.
3. Run `go run . stats` to check there are sightings to show. If it errors or reports nothing,
   fix that first: see "The data" below.
4. Their brief: $ARGUMENTS

   If that's empty, ask what they'd like the site to look like, then wait for the answer. A mood,
   some colours, a site they like or a screenshot pasted in: whatever they have.
5. Design the home page only. Don't ask which pages they'd like or offer more: one page done well
   is the session. Nav links can point at its sections (`/#sightings`). If they ask for another
   page later, "Pages and routes" below says how.
6. Ask one round of questions, in a single AskUserQuestion call, and only those the brief leaves
   open. At most four, from these:
   1. **The feel**: options built from their brief's words, such as "calm and natural", "bold,
      like a poster" or "dark, like a night safari".
   2. **The colours**: two or three sets, each colour named with its hex.
   3. **How the sightings are shown**: photo cards, a map of where they were seen, a list like a
      spreadsheet, or a timeline.
   4. **Which animals** (`multiSelect`): "every animal" first, then a few that suit the brief,
      picked from `go run . animals` and given plain names ("cheetah", "humpback whale"). Skip it
      when the brief already says, such as "big cats". The answer goes into the home page's
      filter: see "The data".
   5. **Extras** (`multiSelect`): the blocks that suit the brief, in the words "Blocks" below
      gives them.

   The look is never preset: build the feel, colours and lettering from their words and from
   design search results, and if they name colours or fonts, use those. The features may come
   from the blocks. When the brief asks for something striking, moving or 3D, a three.js scene
   can be one of the extras, never the only one and never pushed.
7. The block is an hour. After that one round, build the whole look in one go so they see it
   within about ten minutes, then work in small rounds of their choosing. Don't ask a second
   round of questions before they've seen the first result.
8. If they aren't running it yet, suggest `make web` in another terminal, with
   http://localhost:8080 open. It doesn't reload by itself: see "See the site" below.

## Design data

`python3 .claude/skills/ui-ux-pro-max/scripts/search.py "<their words>" --design-system`, run from the app's folder, prints a design system: a
palette, a font pairing, effects and a checklist. Swap `--design-system` for `--domain color` (or
`typography` or `style`) to answer one question instead, or `--domain ux` to look up a UX guideline.
On Windows the command is `python` rather than `python3`.

- It matches keywords against product types, so a result can miss their brief entirely: its
  generic fallback is blue `#2563EB` with orange and a "Hero + Features + CTA" pattern, and odd
  matches happen too, such as a restaurant's browns. When that happens, search once more with
  their brief described as a kind of product, and show them both.
- Its patterns are for landing pages and its implementation advice assumes React. Skip both: this
  is a site of sightings rendered with templ.
- Never paste what it prints. Show the colours by name and hex and the lettering by feel, and
  leave its style, effect and pattern names out unless they ask.
- Once they're happy with a direction, write it to `web/DESIGN.md` so later sessions keep to it:
  the palette and fonts as the CSS custom properties that hold them, spacing, radii, the look of
  each component, and anything they asked to avoid. Record what they chose, not what the search
  printed, and keep it up to date as the design changes. If `web/DESIGN.md` already exists, read
  it before searching and follow it unless they ask for something new.
- Run it from the app's folder, where `.claude/skills/` is, whatever ui-ux-pro-max's own examples
  assume about the repository root.
- If it fails because Python isn't installed, say so and carry on from their brief.

## The data

The sightings live in `workshop.db` in the app folder. It's theirs, and it goes to their fork with
everything else. `go run . stats` reads it:

- **"database migrations are pending"**: run `go run . db migrate up`, then `go run . db seed`.
- **"Nothing to report yet"**: the database is empty, so run `go run . db seed`. It takes a few
  seconds, reads the workshop's bundled copy of GBIF (no network, no key), and is safe to run
  again: sightings already there are skipped.
- **Anything else** (an error about the file, a table or a column): run
  `go run . db migrate status` to see where it stands, and explain in plain words. If seeding
  doesn't fix it, offer `go run . db reset --yes` and then `go run . db seed`. Reset deletes every
  sighting, so ask before running it.

Their `make web` reads the database on every request, so a refresh shows new data without a
restart. Don't use `--live`, `sightings ingest` or `db migrate down`, and never delete or edit
`workshop.db` by hand.

The database holds every animal the app knows. Which ones the home page shows is decided in
`web/handlers/home.go`, by the `sighting.ListFilter` passed to `List`: set `Animals` to the ones they
chose, using the constants in `internal/domain/species/animal.go` (`species.AnimalCheetah`)
that `go run . animals` lists. `Country` and `FromDate` narrow it further, and `Limit` caps it.
Leave `Animals` empty for every animal.

## Pages and routes

Only when they ask for another page. `web.New` in `web/web.go` is the one place that knows a
URL. A page is three things:

1. A templ page in `web/views/pages/<name>.templ`, wrapped in the layout.
2. A handler in `web/handlers/<name>.go`: a function that takes what it needs and returns an
   `http.HandlerFunc`, like `handlers.Home`. It declares the one-method interface it needs beside
   it, the way `home.go` declares `SightingsLister`:

   ```go
   type SightingFinder interface {
   	Find(ctx context.Context, id uuid.UUID) (sighting.Sighting, error)
   }
   ```

   Read path values with `r.PathValue("id")` and parse with `uuid.Parse` (the standard library's
   `uuid`). Answer `http.NotFound` for a bad id or `sighting.ErrNotFound`, and a 500 for anything
   else, logged as `home.go` does.
3. One line in `web.New`, e.g. `mux.Handle("GET /sightings/{id}", handlers.Sighting(logger, sightingsSvc))`.

Data comes from the `*sighting.Service` that `web.New` already has. `List` takes a
`sighting.ListFilter` (`Animals`, `Country`, `FromDate`, `ToDate`, `Limit`), and `Find` takes an
id. For totals and tallies, use `sighting.Summarise`, `sighting.CountBy`, `sighting.CountByYear`
and `sighting.Top` rather than new queries. Species details travel on each sighting. Don't change
`web.New`'s signature: `internal/cmd` calls it.

The header, nav, logo and footer belong in the layout, so every page shares them. Grow
`layout.BaseProps` as needed (a description, the current page for the nav's `aria-current`), and
link between pages with plain `<a href>`s.

## Blocks

`blocks/`, beside this file, holds ready-made pieces to copy into `web/`. `blocks/README.md` lists
them. Read a block's `.md` before using it, and only suggest one when their brief calls for it.
Each line after the name is how to offer it:

- **favicon**: "a logo in the header and on the browser tab". Every site gets this one, so ask
  instead what it should be: an animal, a shape, a letter?
- **swiper**: "a row of photos you swipe through on a phone, side by side on a laptop".
- **lightbox**: "click a photo to see it big".
- **reveal**: "sections fade in as you scroll down to them".
- **threejs**: "a 3D scene, such as a spinning globe". What the scene is, they decide; you build
  it. The block is only the plumbing.
- **weather**: "the weather on the day each animal was seen", from the weather already stored.
- **map**: "a map of where each animal was seen, the dots grouped by area or a glow where there
  are most". Leaflet, with no API key.
- **charts**: "a small chart of sightings per year, and bars for the most-seen animals".
- **filter**: "buttons to narrow the list by animal or country, and a search box". It works in
  the browser, so every sighting is still in the page.
- **transitions**: "a smooth fade from one page to the next", once there's more than one.

Copy the CSS and JS files into `web/assets/`, and the templ from the `.md` into `web/views/`, then
make them look like the rest of their site.

## Keep it building

- Change files under `web/` only: views, assets, `web.go`, and files in `web/handlers/`. The one
  exception is the log line fix in "Start". The `go run .` commands in "The data" are the only
  things you run against the rest of the app.
- `web/handlers/home.go` is theirs to change, including the `ListFilter` that picks which sightings
  the home page shows. Keep the home page at `/` listing sightings through `sightings.List` and
  showing each one's observers (`s.RecordedBy`, already tidied by the service), whether on a card,
  in a table row or in a map popup. The Profiling block measures exactly that.
- Components take domain types directly, or props built from them once. Helpers are plain Go functions in a file beside the `.templ` that uses them.
- Don't comment templ components. A Go helper gets one short line at most, saying what it does.
- Files under `web/assets/` are embedded and served at `/assets/`. The Docker build only runs
  `go build`, so there's no bundler: CSS and scripts are plain files there, and libraries such as
  Leaflet or Swiper load from a CDN. If they want Tailwind, explain it would have to come from a
  CDN too, then let them decide.
- GBIF data is untrusted and patchy. Never pass it to `templ.Raw` or a script's `innerHTML`, send
  URLs through `templ.URL`, and don't build inline styles from it. Expect missing names, photos,
  counts and observers, and keep the empty states.
- Photos come in every size and orientation. Give each image slot a fixed `aspect-ratio` with
  `object-fit: cover`, so they line up however the photo was taken, unless they've asked for a
  masonry-style wall. Keep `loading="lazy"`: the home page renders every sighting at once,
  thousands of them, so whatever you render for one is repeated thousands of times.
- When a round of edits is done, run `make templ` and then `go build ./...`, and fix what fails
  before going on. The generated `_templ.go` files are what gets deployed.
- Modules are vendored. If you `go get` one, run `make vendor` afterwards, or the build refuses it.
- Don't run git. The session commits and pushes to their fork by itself after each of your turns,
  so don't mention it unless they ask.

## See the site

Their `make web` on http://localhost:8080 is the preview. It serves the build from when it
started, stylesheet included, since the assets are embedded. Don't start a server of your own,
and don't stop theirs.

- When a round of edits builds, tell them to restart it (Ctrl+C, then `make web`) and refresh.
- Once they have, `curl -s http://localhost:8080/<page>` shows what a page rendered.
- If they're running `make web-hot`, suggest `make web` for this session instead. templier
  rebuilds while you're still editing, so it can end up serving a stale build.
- Unless you have a browser tool, you can't see the site. After a big visible change, ask them
  for a screenshot and check it against their brief.

## Hand back

Check the work against the checklist the design search printed: contrast, focus states, alt text, a
375px-wide screen, and every nav link going somewhere. Report what you find in plain words, such
as "the grey text on white is hard to read", and fix it.

Then say what changed and what's on the page. They watched the app being built rather than
writing it, so this is where they learn templ: point at one or two of the files they now own and
explain it in Go terms. A component is a Go function returning a `templ.Component`, its props are
the parameters, `for` and `if` are Go's own, and the layout's `{ children... }` is where each page
slots in. Keep it to a few lines and offer more if they want it.

Ask what they'd like to change next. Their work is already on their fork, which is what the
workshop looks at afterwards.
