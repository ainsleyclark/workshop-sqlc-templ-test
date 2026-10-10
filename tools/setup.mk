tools: # Build the Go tools from vendor/ (nothing is downloaded) and the app, as workshop on your PATH
	$(info Installing dependencies...)
	$(info )
	go install github.com/a-h/templ/cmd/templ
	go -C tools/gopls install golang.org/x/tools/gopls
	go tool sqlc version
	go install github.com/romshark/templier
	go install .
	go build ./...
.PHONY: tools

tools-online: # The same tools at the same versions from the internet
	go install github.com/a-h/templ/cmd/templ@v0.3.1020
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
	go install github.com/romshark/templier@v0.13.0
	go install golang.org/x/tools/gopls@v0.23.0
.PHONY: tools-online

setup: export GOTOOLCHAIN = local
setup: tools # Build the tools; run once after cloning your fork
	$(info $(SETUP_BANNER))
.PHONY: setup

define SETUP_BANNER

           .
          ":"
        ___:____     |"\/"|
      ,'        `.    \  /
      |  O        \___/  |
    ~^~^~^~^~^~^~^~^~^~^~^~^~

  Welcome to the workshop!

  Setup done. Fill the database, then see the site:

    1. make run, then db -> migrate -> up
    2. make run, then sightings -> ingest (500 is plenty)
    3. make web, then open http://localhost:8080

  Shortcut: go run . db seed does 1 and 2 in one go.

endef
