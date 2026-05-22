package intrinsics

import (
	"sort"
	"strings"

	"baziclang/internal/ast"
)

func MatchesProgramTypeName(t ast.Type, target string) bool {
	return string(t) == target || strings.HasSuffix(string(t), "__"+target)
}

func ResolveProgramTypeName(p *ast.Program, sourceName string) string {
	if p == nil {
		return sourceName
	}
	for _, decl := range p.Decls {
		switch d := decl.(type) {
		case *ast.StructDecl:
			if d.Name == sourceName {
				return FirstNonEmpty(d.InternalName, d.Name)
			}
		case *ast.InterfaceDecl:
			if d.Name == sourceName {
				return FirstNonEmpty(d.InternalName, d.Name)
			}
		case *ast.EnumDecl:
			if d.Name == sourceName {
				return FirstNonEmpty(d.InternalName, d.Name)
			}
		}
	}
	return sourceName
}

func CollectHTTPHandlers(p *ast.Program) []HTTPHandlerSpec {
	handlers := []HTTPHandlerSpec{}
	if p == nil {
		return handlers
	}
	for _, d := range p.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if h, ok := ParseHTTPHandler(fn); ok {
			handlers = append(handlers, h)
		}
	}
	if len(handlers) > 1 {
		sort.Slice(handlers, func(i, j int) bool {
			if handlers[i].Method == handlers[j].Method {
				return handlers[i].FuncName < handlers[j].FuncName
			}
			return handlers[i].Method < handlers[j].Method
		})
	}
	return handlers
}

func ParseHTTPHandler(fn *ast.FuncDecl) (HTTPHandlerSpec, bool) {
	if fn == nil || len(fn.Params) != 1 {
		return HTTPHandlerSpec{}, false
	}
	if !MatchesProgramTypeName(fn.Params[0].Type, "ServerRequest") {
		return HTTPHandlerSpec{}, false
	}
	if !MatchesProgramTypeName(fn.ReturnType, "ServerResponse") {
		return HTTPHandlerSpec{}, false
	}
	parts := strings.Split(fn.Name, "_")
	if len(parts) < 2 {
		return HTTPHandlerSpec{}, false
	}
	method := strings.ToUpper(parts[0])
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
	default:
		return HTTPHandlerSpec{}, false
	}
	if len(parts) == 2 && parts[1] == "root" {
		return HTTPHandlerSpec{Method: method, Segments: nil, FuncName: fn.Name}, true
	}
	segments := []HTTPRouteSegmentSpec{}
	for i := 1; i < len(parts); {
		if parts[i] == "p" {
			if i+1 >= len(parts) || parts[i+1] == "" {
				return HTTPHandlerSpec{}, false
			}
			segments = append(segments, HTTPRouteSegmentSpec{Param: parts[i+1], IsParam: true})
			i += 2
			continue
		}
		if parts[i] == "" {
			return HTTPHandlerSpec{}, false
		}
		segments = append(segments, HTTPRouteSegmentSpec{Literal: parts[i]})
		i++
	}
	return HTTPHandlerSpec{Method: method, Segments: segments, FuncName: fn.Name}, true
}

func HTTPRoutePattern(h HTTPHandlerSpec) string {
	if len(h.Segments) == 0 {
		return "/"
	}
	parts := make([]string, 0, len(h.Segments))
	for _, seg := range h.Segments {
		if seg.IsParam {
			parts = append(parts, ":"+seg.Param)
		} else {
			parts = append(parts, seg.Literal)
		}
	}
	return "/" + strings.Join(parts, "/")
}
