package langsurface

import "baziclang/internal/intrinsics"

type SurfaceSymbol struct {
	Name           string
	Hover          string
	CompletionKind int
}

func SurfaceSymbols() []SurfaceSymbol {
	items := make([]SurfaceSymbol, 0, len(keywordSpecs)+len(intrinsics.SurfaceFunctionSpecs()))
	for _, spec := range KeywordSpecs() {
		items = append(items, SurfaceSymbol{
			Name:           spec.Name,
			Hover:          spec.Hover,
			CompletionKind: spec.CompletionKind,
		})
	}
	for _, spec := range intrinsics.SurfaceFunctionSpecs() {
		items = append(items, SurfaceSymbol{
			Name:           spec.Name,
			Hover:          spec.Hover(),
			CompletionKind: 3,
		})
	}
	return items
}

func LookupSurfaceSymbol(name string) (SurfaceSymbol, bool) {
	if spec, ok := LookupKeyword(name); ok {
		return SurfaceSymbol{
			Name:           spec.Name,
			Hover:          spec.Hover,
			CompletionKind: spec.CompletionKind,
		}, true
	}
	if spec, ok := intrinsics.LookupSurfaceFunction(name); ok {
		return SurfaceSymbol{
			Name:           spec.Name,
			Hover:          spec.Hover(),
			CompletionKind: 3,
		}, true
	}
	return SurfaceSymbol{}, false
}
