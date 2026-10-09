# The workshop repository this copy was forked from, as owner/repo. make publish
# in the maintainer repository rewrites this line for each event.
UPSTREAM := ainsleyclark/workshop-sqlc-templ-test

setup: # Install the toolchain and point upstream at the workshop repository; run once after cloning your fork
	sh tools/setup/setup.sh $(UPSTREAM)
.PHONY: setup

update: # Pull the instructor's latest changes from the workshop repository into your fork
	git pull --rebase upstream main
.PHONY: update

push: # Commit everything and push it to your fork.
	@[ "$$(git rev-parse --show-toplevel 2>/dev/null)" = "$$(pwd -P)" ] || { echo "make push runs from the root of your fork"; exit 1; }
	git add .
	-git commit -q -m "feat: Attendee Push"
	git push -q
.PHONY: push

run: # Run the program
	go run .
.PHONY: run

run-live: # Run the program against the live GBIF and Open-Meteo APIs instead of the mock; they're rate limited
	go run . --live
.PHONY: run-live

web: # Serve the front-end
	go run main.go web
.PHONY: web

web-hot: # Serve hot reload with templ
	CLICOLOR_FORCE=1 go tool templier --config ./templier.yml
.PHONY: web-hot

key: # Save the workshop's Anthropic key for Claude Code, then run `claude` here and type /design-site
	@py=$$(command -v python3 || command -v python) || { echo "make key needs Python 3"; exit 1; }; \
	printf 'Paste the workshop key from Slack, or press Enter to use your own Claude login: '; \
	read -r key; \
	[ -n "$$key" ] || { echo "No key saved; Claude Code will use your own login."; exit 0; }; \
	KEY="$$key" "$$py" -c 'import json, os, pathlib; p = pathlib.Path(".claude/settings.local.json"); s = json.loads(p.read_text()) if p.exists() else {}; s.setdefault("env", {})["ANTHROPIC_API_KEY"] = os.environ["KEY"].strip().strip("\x27\x22"); p.parent.mkdir(exist_ok=True); p.write_text(json.dumps(s, indent=2) + "\n")' && \
	echo "Saved to .claude/settings.local.json. Start Claude Code here with: claude, then type /design-site"
.PHONY: key

mock: # Serve the mock GBIF and Open-Meteo APIs on :8082.
	go run ./cmd/mockapi
.PHONY: mock

mock-scrape: # Refresh the mock's GBIF data from the live API.
	go run ./cmd/mockapi scrape
.PHONY: mock-scrape

themes: # Serve the themed examples on :8083
	go run ./cmd/themes
.PHONY: themes

format: # Run gofmt and templ fmt
	go fmt ./...
	go tool templ fmt .
.PHONY: format

test: # Run all tests
	go clean -testcache && go test ./... -coverprofile=coverage.out -covermode=atomic
.PHONY: test

test-race: # Run all tests with race
	go clean -testcache && go test -race ./... -coverprofile=coverage.out -covermode=atomic
.PHONY: test-race

load: # Load test the home page for 20 seconds while make web runs
	@command -v hey >/dev/null 2>&1 || go install github.com/rakyll/hey@latest
	hey -z 10s -c 10 http://localhost:8080/
.PHONY: load

load-pprof: # Load test the home page for 20 seconds while make web runs, saving CPU and allocation profiles here
	@command -v hey >/dev/null 2>&1 || go install github.com/rakyll/hey@latest
	@curl -s -o ./cpu.pprof "http://localhost:6060/debug/pprof/profile?seconds=10" & \
	hey -z 20s -c 10 http://localhost:8080/; \
	wait; \
	curl -s -o ./allocs.pprof http://localhost:6060/debug/pprof/allocs; \
	echo; \
	echo "Profiles saved, open one with:"; \
	echo "  go tool pprof -http=: ./cpu.pprof"; \
	echo "  go tool pprof -http=: -sample_index=alloc_space ./allocs.pprof"
.PHONY: load-pprof

clean:
	go fmt ./...
	go fix ./...
.PHONY: clean

gen: # Runs all //go:generate.
	go generate ./...
.PHONY: gen

templ: # Runs templ generate.
	go tool templ generate ./...
.PHONY: templ

sqlc: # Regenerate sqlc output.
	CGO_ENABLED=0 go tool sqlc generate
	go build ./...
.PHONY: sqlc

vendor: # Re-vendor after go get, here and in tools/gopls.
	go mod tidy
	go mod vendor
	cd tools/gopls && go mod tidy && go mod vendor
.PHONY: vendor

kill-ports: # Kill anything listening on the ports this app uses.
	@for port in 8080 8081 8082 8083; do \
		pid=$$(lsof -ti tcp:$$port); \
		if [ -n "$$pid" ]; then \
			echo "Killing $$pid on :$$port"; \
			kill -9 $$pid; \
		fi; \
	done
.PHONY: kill-ports

help: # Display this help
	$(Q) awk 'BEGIN {FS = ":.*#"; printf "Usage: make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?#/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.PHONY: help
