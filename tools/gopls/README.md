# gopls

The Go language server, for editors that use it (VS Code, Zed, Neovim; GoLand has its own).
`make setup` installs it to `~/go/bin` from the vendored source here, so it needs no network.

It's a module of its own rather than a `tool` line in the app's `go.mod`: gopls only builds
against the exact `golang.org/x/tools` it was released with, and the app's dependencies
(templier, modernc.org/libc) need a newer one. In one module they'd share the newer one and gopls
would fail to compile.

To upgrade, from this folder:

```sh
GOFLAGS=-mod=mod go get -tool golang.org/x/tools/gopls@vX.Y.Z
go mod tidy && go mod vendor
```
