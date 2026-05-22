package backendmeta

import (
	"strings"

	"baziclang/internal/mir"
)

func ResolveProgramTypeName(p *mir.Program, sourceName string) string {
	if p == nil || sourceName == "" {
		return sourceName
	}
	for _, decl := range p.Decls {
		name, ok := declTypeName(decl)
		if !ok {
			continue
		}
		if name == sourceName || strings.HasSuffix(name, "__"+sourceName) {
			return name
		}
	}
	return sourceName
}

func HasProgramTypeName(p *mir.Program, sourceName string) bool {
	if p == nil || sourceName == "" {
		return false
	}
	for _, decl := range p.Decls {
		name, ok := declTypeName(decl)
		if !ok {
			continue
		}
		if name == sourceName || strings.HasSuffix(name, "__"+sourceName) {
			return true
		}
	}
	return false
}

func declTypeName(decl mir.Decl) (string, bool) {
	switch d := decl.(type) {
	case *mir.StructDecl:
		return d.Name, true
	case *mir.InterfaceDecl:
		return d.Name, true
	case *mir.EnumDecl:
		return d.Name, true
	default:
		return "", false
	}
}
