package main

import "workshop/internal/cmd"

//go:generate env CGO_ENABLED=0 go tool sqlc generate

func main() {
	cmd.Run()
}
