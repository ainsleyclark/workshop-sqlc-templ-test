package cmd

import (
	"github.com/urfave/cli/v3"
)

// countryFlag is the shared country selector. The usage differs per
// command, since one filters saved sightings and the other fetches them.
func countryFlag(usage string) cli.Flag {
	return &cli.StringFlag{
		Name:    "country",
		Aliases: []string{"c"},
		Usage:   usage,
	}
}

// limitFlag caps how many records a command fetches or shows.
func limitFlag(value int, usage string) cli.Flag {
	return &cli.IntFlag{
		Name:    "limit",
		Aliases: []string{"n"},
		Usage:   usage,
		Value:   value,
	}
}
