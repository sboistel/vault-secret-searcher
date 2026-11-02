package main

import (
	"context"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
)

func main() {
	var cli CLI

	log.SetLevel(log.InfoLevel)

	pctx := kong.Parse(&cli,
		kong.Name("vault-secret-searcher"),
		kong.Description("Search Vault KV secrets for matching keys or values\n\nExamples:\n  vault-secret-searcher -k redis\n  vault-secret-searcher -p secret/infra/ -v \"secret_value\""),
	)

	if cli.Debug {
		log.SetLevel(log.DebugLevel)
	}

	pctx.BindTo(context.Background(), (*context.Context)(nil))
	err := pctx.Run(context.Background())
	pctx.FatalIfErrorf(err)
}
