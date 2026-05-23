package releasecontract

import "strings"

const (
	ReleaseTrackAlpha        = "alpha"
	GoBackendReleaseStatus   = "stable release path"
	LLVMBackendReleaseStatus = "experimental"
)

var alphaStableStdModules = []string{
	"io", "fs", "time", "json", "http", "crypto", "base64", "collections", "os", "path",
}

var alphaExperimentalStdModules = []string{
	"db", "auth", "jwt", "session", "desktop", "web", "ui", "sql", "validate",
}

func AlphaStableStdModules() []string {
	return append([]string(nil), alphaStableStdModules...)
}

func AlphaExperimentalStdModules() []string {
	return append([]string(nil), alphaExperimentalStdModules...)
}

func JoinModules(mods []string) string {
	return strings.Join(mods, ", ")
}
