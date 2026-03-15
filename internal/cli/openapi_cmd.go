package cli

import (
	"flag"
	"fmt"
	"strings"

	"baziclang/internal/openapi"
)

func openapiCmd(binaryName string, args []string) int {
	fs := flag.NewFlagSet("openapi", flag.ExitOnError)
	routes := fs.String("routes", "", "bazic routes file")
	models := fs.String("models", "", "bazic models file")
	out := fs.String("out", "openapi.json", "output file")
	title := fs.String("title", "Bazic API", "API title")
	version := fs.String("version", "v1", "API version")
	_ = fs.Parse(args)
	if strings.TrimSpace(*routes) == "" {
		return die("--routes is required")
	}
	parsed, err := openapi.ParseRoutes(*routes)
	if err != nil {
		return die(err.Error())
	}
	schemas := map[string]openapi.Schema{}
	if strings.TrimSpace(*models) != "" {
		s, err := openapi.SchemaFromModels(*models)
		if err != nil {
			return die(err.Error())
		}
		schemas = s
	}
	spec := openapi.Generate(*title, *version, parsed, schemas)
	if err := openapi.Write(spec, *out); err != nil {
		return die(err.Error())
	}
	fmt.Printf("Wrote %s\n", *out)
	return 0
}
