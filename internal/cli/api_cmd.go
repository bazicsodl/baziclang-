package cli

import (
	"flag"
	"fmt"
	"strings"

	"baziclang/internal/api"
	"baziclang/internal/openapi"
)

func apiCmd(binaryName string, args []string) int {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	routes := fs.String("routes", "", "bazic routes file")
	models := fs.String("models", "", "bazic models file")
	out := fs.String("out", "handlers.bz", "output handlers file")
	_ = fs.Parse(args)
	if strings.TrimSpace(*routes) == "" {
		return die("--routes is required")
	}
	if strings.TrimSpace(*models) == "" {
		return die("--models is required")
	}
	r, err := openapi.ParseRoutes(*routes)
	if err != nil {
		return die(err.Error())
	}
	endpoints := make([]api.Endpoint, 0, len(r))
	for _, e := range r {
		endpoints = append(endpoints, api.Endpoint{Method: e.Method, Path: e.Path, Func: e.Func})
	}
	code, err := api.GenerateHandlers(*models, endpoints)
	if err != nil {
		return die(err.Error())
	}
	if err := api.WriteHandlers(*out, code); err != nil {
		return die(err.Error())
	}
	fmt.Printf("Wrote %s\n", *out)
	return 0
}
