package backendmeta

import (
	"strings"

	"baziclang/internal/mir"
)

type GoRuntimeHelperSection string

const (
	GoRuntimeHelperCore       GoRuntimeHelperSection = "core"
	GoRuntimeHelperStdRuntime GoRuntimeHelperSection = "std_runtime"
	GoRuntimeHelperJSONWeb    GoRuntimeHelperSection = "json_web"
	GoRuntimeHelperHTTP       GoRuntimeHelperSection = "http"
	GoRuntimeHelperCrypto     GoRuntimeHelperSection = "crypto"
	GoRuntimeHelperHTTPServe  GoRuntimeHelperSection = "http_serve"
	GoRuntimeHelperSession    GoRuntimeHelperSection = "session"
	GoRuntimeHelperDB         GoRuntimeHelperSection = "db"
)

type GoRuntimeSurfaceMeta struct {
	Target            string
	Imports           []string
	HasSessionPrelude bool
	HelperSections    []GoRuntimeHelperSection
	SessionPrelude    string
	HelperSnippets    []string
	Prelude           string
}

type GoRuntimePlan = GoRuntimeSurfaceMeta

func OrderedGoRuntimeHelperSections() []GoRuntimeHelperSection {
	return []GoRuntimeHelperSection{
		GoRuntimeHelperCore,
		GoRuntimeHelperStdRuntime,
		GoRuntimeHelperJSONWeb,
		GoRuntimeHelperHTTP,
		GoRuntimeHelperCrypto,
		GoRuntimeHelperHTTPServe,
		GoRuntimeHelperSession,
		GoRuntimeHelperDB,
	}
}

func CollectGoRuntimeSurfaceMeta(meta ProgramRuntimeMeta, target string) GoRuntimeSurfaceMeta {
	target = strings.ToLower(strings.TrimSpace(target))
	imports := collectGoRuntimeImports(meta.Features, target)

	sections := collectGoRuntimeHelperSections(meta.Features)

	sessionPrelude := ""
	if HasRuntimeFeature(meta.Features, RuntimeFeatureGoSession) {
		sessionPrelude = "type __bazic_session_entry struct { UserID string; ExpiresAt time.Time }\n" +
			"var __bazic_session_mu sync.Mutex\n" +
			"var __bazic_session_store = map[string]__bazic_session_entry{}\n\n"
	}

	helpers := collectGoRuntimeHelperSnippets(meta, target, sections)

	surface := GoRuntimeSurfaceMeta{
		Target:            target,
		Imports:           imports,
		HasSessionPrelude: HasRuntimeFeature(meta.Features, RuntimeFeatureGoSession),
		HelperSections:    sections,
		SessionPrelude:    sessionPrelude,
		HelperSnippets:    helpers,
	}
	surface.Prelude = RenderGoPrelude(surface)
	return surface
}

func BuildGoRuntimePlan(meta ProgramRuntimeMeta, target string) GoRuntimePlan {
	return CollectGoRuntimeSurfaceMeta(meta, target)
}

type GoProgramPlan struct {
	Shape       ProgramShapeMeta
	RuntimePlan GoRuntimePlan
	Prelude     string
}

func CollectGoProgramPlan(p *mir.Program, target string) GoProgramPlan {
	shape := CollectProgramShapeMeta(p)
	runtimePlan := CollectGoRuntimeSurfaceMeta(shape.Runtime, target)
	return GoProgramPlan{
		Shape:       shape,
		RuntimePlan: runtimePlan,
		Prelude:     runtimePlan.Prelude,
	}
}

func RenderGoPrelude(plan GoRuntimePlan) string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString("import (\n")
	for _, imp := range plan.Imports {
		b.WriteString("\t")
		b.WriteString(imp)
		b.WriteString("\n")
	}
	b.WriteString(")\n\n")
	b.WriteString(plan.SessionPrelude)
	for _, helper := range plan.HelperSnippets {
		b.WriteString(helper)
	}
	return b.String()
}
