package releasecontract

import "strings"

const (
	ReleaseTrackAlpha        = "alpha"
	GoBackendReleaseStatus   = "stable release path"
	LLVMBackendReleaseStatus = "experimental"
)

type StdlibTier string

const (
	StdlibTierStable       StdlibTier = "stable"
	StdlibTierExperimental StdlibTier = "experimental"
)

type StdlibModule struct {
	Name string
	Tier StdlibTier
}

var alphaStdlibModules = []StdlibModule{
	{Name: "io", Tier: StdlibTierStable},
	{Name: "fs", Tier: StdlibTierStable},
	{Name: "time", Tier: StdlibTierStable},
	{Name: "json", Tier: StdlibTierStable},
	{Name: "http", Tier: StdlibTierStable},
	{Name: "crypto", Tier: StdlibTierStable},
	{Name: "base64", Tier: StdlibTierStable},
	{Name: "collections", Tier: StdlibTierStable},
	{Name: "os", Tier: StdlibTierStable},
	{Name: "path", Tier: StdlibTierStable},
	{Name: "db", Tier: StdlibTierExperimental},
	{Name: "auth", Tier: StdlibTierExperimental},
	{Name: "jwt", Tier: StdlibTierExperimental},
	{Name: "session", Tier: StdlibTierExperimental},
	{Name: "desktop", Tier: StdlibTierExperimental},
	{Name: "web", Tier: StdlibTierExperimental},
	{Name: "ui", Tier: StdlibTierExperimental},
	{Name: "sql", Tier: StdlibTierExperimental},
	{Name: "validate", Tier: StdlibTierExperimental},
}

func AlphaStdlibModules() []StdlibModule {
	return append([]StdlibModule(nil), alphaStdlibModules...)
}

func AlphaStableStdModules() []string {
	return stdlibModuleNamesByTier(StdlibTierStable)
}

func AlphaExperimentalStdModules() []string {
	return stdlibModuleNamesByTier(StdlibTierExperimental)
}

func JoinModules(mods []string) string {
	return strings.Join(mods, ", ")
}

func stdlibModuleNamesByTier(tier StdlibTier) []string {
	out := make([]string, 0, len(alphaStdlibModules))
	for _, mod := range alphaStdlibModules {
		if mod.Tier == tier {
			out = append(out, mod.Name)
		}
	}
	return out
}
