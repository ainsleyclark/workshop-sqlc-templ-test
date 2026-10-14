# The workshop repository this copy was forked from, as owner/repo. make publish
# in the maintainer repository rewrites this line for each event.
UPSTREAM := ainsleyclark/workshop-sqlc-templ-test

include tools/setup.mk

run: # Run the program
	go run .
.PHONY: run

run-live: # Run the program against the live GBIF and Open-Meteo APIs instead of the mock; they're rate limited
	go run . --live
.PHONY: run-live

web: # Serve the front-end
	go run main.go web
.PHONY: web

web-hot: export CLICOLOR_FORCE = 1
web-hot: # Serve hot reload with templ
	go tool templier --config ./templier.yml
.PHONY: web-hot

key: # Save the workshop's Anthropic key for Claude Code, then run `claude` here and type /design-site
	go run ./tools/key
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

load: # Load test the home page for 10 seconds while make web runs
	go tool hey -z 10s -c 10 http://localhost:8080/
.PHONY: load

# -j2 runs the CPU profile and the load at the same time.
load-pprof: # Load test the home page for 10 seconds while make web runs, saving CPU and allocation profiles here
	$(MAKE) --no-print-directory -j2 profile-cpu load
	go tool pprof -proto -output allocs.pprof http://localhost:6060/debug/pprof/allocs
	@echo Profiles saved. Open one with:
	@echo   go tool pprof -http=: ./cpu.pprof
	@echo   go tool pprof -http=: -sample_index=alloc_space ./allocs.pprof
.PHONY: load-pprof

profile-cpu:
	go tool pprof -proto -output cpu.pprof "http://localhost:6060/debug/pprof/profile?seconds=10"
.PHONY: profile-cpu

update: # Pull the instructor's latest changes from the workshop repository into your fork
	git pull --rebase https://github.com/$(UPSTREAM).git main
.PHONY: update

push: # Commit everything and push it to your fork.
	@$(if $(wildcard .git),,$(error make push runs from the root of your fork))
	git add .
	-git commit -q -m "feat: Attendee Push"
	git push -q
.PHONY: push

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
	go tool sqlc generate
	go build ./...
.PHONY: sqlc

vendor: # Re-vendor after go get, here and in tools/gopls.
	go mod tidy
	go mod vendor
	cd tools/gopls && go mod tidy && go mod vendor
.PHONY: vendor

kill-ports: # Kill anything listening on the ports this app uses (macOS/Linux).
	@for port in 8080 8081 8082 8083; do \
		pid=$$(lsof -ti tcp:$$port); \
		if [ -n "$$pid" ]; then \
			echo "Killing $$pid on :$$port"; \
			kill -9 $$pid; \
		fi; \
	done
.PHONY: kill-ports

help: # Display this help (macOS/Linux)
	$(Q) awk 'BEGIN {FS = ":.*#"; printf "Usage: make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?#/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.PHONY: help
