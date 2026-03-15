package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Spec struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}

type Schema struct {
	Ref        string            `json:"$ref,omitempty"`
	Type       string            `json:"type,omitempty"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Items      *Schema           `json:"items,omitempty"`
}

type PathItem map[string]Operation

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Schema   Schema `json:"schema"`
}

type RequestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]MediaType `json:"content"`
}

type MediaType struct {
	Schema Schema `json:"schema"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type Route struct {
	Method string
	Path   string
	Params []string
	Func   string
}

func ParseRoutes(bzPath string) ([]Route, error) {
	data, err := os.ReadFile(bzPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	routes := []Route{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "fn ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, "fn "))
		idx := strings.Index(name, "(")
		if idx == -1 {
			continue
		}
		name = strings.TrimSpace(name[:idx])
		parts := strings.Split(name, "_")
		if len(parts) < 2 {
			continue
		}
		method := strings.ToUpper(parts[0])
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		default:
			continue
		}
		path, params, ok := routeFromParts(parts)
		if !ok {
			continue
		}
		routes = append(routes, Route{Method: method, Path: path, Params: params, Func: name})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return routes, nil
}

func routeFromParts(parts []string) (string, []string, bool) {
	if len(parts) == 2 && parts[1] == "root" {
		return "/", nil, true
	}
	segments := []string{}
	params := []string{}
	for i := 1; i < len(parts); {
		if parts[i] == "p" {
			if i+1 >= len(parts) {
				return "", nil, false
			}
			name := parts[i+1]
			params = append(params, name)
			segments = append(segments, "{"+name+"}")
			i += 2
			continue
		}
		segments = append(segments, parts[i])
		i++
	}
	return "/" + strings.Join(segments, "/"), params, true
}

func Generate(title string, version string, routes []Route, schemas map[string]Schema) Spec {
	paths := map[string]PathItem{}
	for _, r := range routes {
		item := paths[r.Path]
		if item == nil {
			item = PathItem{}
		}
		op := Operation{
			Summary:   r.Func,
			Responses: map[string]Response{"200": {Description: "ok"}},
		}
		if len(r.Params) > 0 {
			params := []Parameter{}
			for _, p := range r.Params {
				params = append(params, Parameter{Name: p, In: "path", Required: true, Schema: Schema{Type: "string"}})
			}
			op.Parameters = params
		}
		if !strings.HasPrefix(r.Func, "public_") {
			op.Responses["401"] = Response{Description: "unauthorized"}
		}
		op.Responses["400"] = Response{Description: "bad request"}
		modelName, mode := inferModelForRoute(r.Path, schemas, r.Method)
		if modelName != "" {
			ref := Schema{Ref: "#/components/schemas/" + modelName}
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
				op.RequestBody = &RequestBody{
					Required: true,
					Content: map[string]MediaType{
						"application/json": {Schema: ref},
					},
				}
			}
			if mode == "list" {
				op.Responses["200"] = Response{
					Description: "ok",
					Content: map[string]MediaType{
						"application/json": {Schema: Schema{Type: "array", Items: &ref}},
					},
				}
				op.Parameters = append(op.Parameters, Parameter{Name: "limit", In: "query", Required: false, Schema: Schema{Type: "integer"}})
				op.Parameters = append(op.Parameters, Parameter{Name: "offset", In: "query", Required: false, Schema: Schema{Type: "integer"}})
				op.Parameters = append(op.Parameters, Parameter{Name: "order", In: "query", Required: false, Schema: Schema{Type: "string"}})
				op.Parameters = append(op.Parameters, Parameter{Name: "dir", In: "query", Required: false, Schema: Schema{Type: "string"}})
			} else if mode == "single" {
				op.Responses["200"] = Response{
					Description: "ok",
					Content: map[string]MediaType{
						"application/json": {Schema: ref},
					},
				}
				op.Responses["404"] = Response{Description: "not found"}
			}
			if r.Method == "POST" {
				op.Responses["201"] = op.Responses["200"]
			}
			if r.Method == "DELETE" {
				op.Responses["204"] = Response{Description: "no content"}
				delete(op.Responses, "200")
			}
		}
		item[strings.ToLower(r.Method)] = op
		paths[r.Path] = item
	}
	return Spec{
		OpenAPI:    "3.0.0",
		Info:       Info{Title: title, Version: version},
		Paths:      paths,
		Components: Components{Schemas: schemas},
	}
}

func Write(spec Spec, path string) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func SchemaFromModels(modelsPath string) (map[string]Schema, error) {
	data, err := os.ReadFile(modelsPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	schemas := map[string]Schema{}
	var current string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "struct ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				current = strings.TrimSpace(parts[1])
				schemas[current] = Schema{Type: "object", Properties: map[string]Schema{}}
			}
			continue
		}
		if line == "}" {
			current = ""
			continue
		}
		if current == "" {
			continue
		}
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			name := strings.TrimSpace(strings.TrimSuffix(parts[0], ";"))
			typePart := strings.TrimSpace(strings.TrimSuffix(parts[1], ";"))
			schemas[current].Properties[name] = Schema{Type: mapOpenAPIType(typePart)}
		}
	}
	return schemas, nil
}

func inferModelForRoute(path string, schemas map[string]Schema, method string) (string, string) {
	if len(schemas) == 0 {
		return "", ""
	}
	segments := []string{}
	for _, seg := range strings.Split(path, "/") {
		if seg == "" || strings.HasPrefix(seg, "{") {
			continue
		}
		segments = append(segments, seg)
	}
	if len(segments) == 0 {
		return "", ""
	}
	last := segments[len(segments)-1]
	keys := map[string]string{}
	for name := range schemas {
		lower := strings.ToLower(name)
		keys[lower] = name
		keys[lower+"s"] = name
	}
	if model, ok := keys[last]; ok {
		if strings.Contains(path, "{") {
			return model, "single"
		}
		if method == "GET" {
			return model, "list"
		}
		return model, "single"
	}
	if len(segments) >= 2 {
		prev := segments[len(segments)-2]
		if model, ok := keys[prev]; ok {
			if strings.Contains(path, "{") {
				return model, "single"
			}
		}
	}
	return "", ""
}

func mapOpenAPIType(t string) string {
	t = strings.TrimSpace(t)
	if strings.HasPrefix(t, "Option[") {
		t = strings.TrimSuffix(strings.TrimPrefix(t, "Option["), "]")
	}
	switch t {
	case "int":
		return "integer"
	case "float":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

func ExampleUsage() string {
	return fmt.Sprintf("%s", "")
}
