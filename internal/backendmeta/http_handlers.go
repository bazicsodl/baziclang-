package backendmeta

import (
	"sort"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/intrinsics"
	"baziclang/internal/mir"
	baztypes "baziclang/internal/types"
)

func CollectHTTPHandlers(p *mir.Program) []intrinsics.HTTPHandlerSpec {
	handlers := []intrinsics.HTTPHandlerSpec{}
	if p == nil {
		return handlers
	}
	for _, d := range p.Decls {
		fn, ok := d.(*mir.FuncDecl)
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

func ParseHTTPHandler(fn *mir.FuncDecl) (intrinsics.HTTPHandlerSpec, bool) {
	if fn == nil || len(fn.Params) != 1 {
		return intrinsics.HTTPHandlerSpec{}, false
	}
	if !matchesProgramTypeName(fn.Params[0].Type, "ServerRequest") {
		return intrinsics.HTTPHandlerSpec{}, false
	}
	if !matchesProgramTypeName(fn.ReturnType, "ServerResponse") {
		return intrinsics.HTTPHandlerSpec{}, false
	}
	parts := strings.Split(fn.Name, "_")
	if len(parts) < 2 {
		return intrinsics.HTTPHandlerSpec{}, false
	}
	method := strings.ToUpper(parts[0])
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
	default:
		return intrinsics.HTTPHandlerSpec{}, false
	}
	if len(parts) == 2 && parts[1] == "root" {
		return intrinsics.HTTPHandlerSpec{Method: method, Segments: nil, FuncName: fn.Name}, true
	}
	segments := []intrinsics.HTTPRouteSegmentSpec{}
	for i := 1; i < len(parts); {
		if parts[i] == "p" {
			if i+1 >= len(parts) || parts[i+1] == "" {
				return intrinsics.HTTPHandlerSpec{}, false
			}
			segments = append(segments, intrinsics.HTTPRouteSegmentSpec{Param: parts[i+1], IsParam: true})
			i += 2
			continue
		}
		if parts[i] == "" {
			return intrinsics.HTTPHandlerSpec{}, false
		}
		segments = append(segments, intrinsics.HTTPRouteSegmentSpec{Literal: parts[i]})
		i++
	}
	return intrinsics.HTTPHandlerSpec{Method: method, Segments: segments, FuncName: fn.Name}, true
}

func matchesProgramTypeName(t baztypes.Type, target string) bool {
	at := baztypes.ToAST(t)
	return at == ast.Type(target) || strings.HasSuffix(string(at), "__"+target)
}
