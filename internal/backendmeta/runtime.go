package backendmeta

import (
	"baziclang/internal/intrinsics"
	"baziclang/internal/mir"
)

type ProgramRuntimeMeta struct {
	Capabilities RuntimeCapabilityMeta
	Types        RuntimeTypeMeta
	Routes       RuntimeRouteMeta
	Features     RuntimeFeatureSurfaceMeta
}

type RuntimeFeature string

const (
	RuntimeFeatureGoCore            RuntimeFeature = "go_core"
	RuntimeFeatureGoStdRuntime      RuntimeFeature = "go_std_runtime"
	RuntimeFeatureGoJSONWeb         RuntimeFeature = "go_json_web"
	RuntimeFeatureGoHTTP            RuntimeFeature = "go_http"
	RuntimeFeatureGoCrypto          RuntimeFeature = "go_crypto"
	RuntimeFeatureGoHTTPServe       RuntimeFeature = "go_http_serve"
	RuntimeFeatureGoSession         RuntimeFeature = "go_session"
	RuntimeFeatureGoDB              RuntimeFeature = "go_db"
	RuntimeFeatureGoImportHMAC      RuntimeFeature = "go_import_hmac"
	RuntimeFeatureGoImportBcrypt    RuntimeFeature = "go_import_bcrypt"
	RuntimeFeatureGoImportSync      RuntimeFeature = "go_import_sync"
	RuntimeFeatureLLVMStringGlobals RuntimeFeature = "llvm_string_globals"
	RuntimeFeatureLLVMRouteTable    RuntimeFeature = "llvm_route_table"
	RuntimeFeatureLLVMStringRuntime RuntimeFeature = "llvm_string_runtime"
	RuntimeFeatureLLVMBuiltin       RuntimeFeature = "llvm_builtin"
	RuntimeFeatureLLVMAnyRuntime    RuntimeFeature = "llvm_any_runtime"
	RuntimeFeatureLLVMStdDecls      RuntimeFeature = "llvm_std_decls"
	RuntimeFeatureLLVMParseInt      RuntimeFeature = "llvm_parse_int"
	RuntimeFeatureLLVMParseFloat    RuntimeFeature = "llvm_parse_float"
)

type RuntimeFeatureSurfaceMeta struct {
	Ordered []RuntimeFeature
}

type RuntimeCapabilityMeta struct {
	HasErrorType      bool
	HasHTTPHandlers   bool
	HasHTTPResponse   bool
	NeedsSession      bool
	NeedsDB           bool
	NeedsBcrypt       bool
	NeedsJWT          bool
	NeedsHMAC         bool
	NeedsHTTPServeApp bool
	NeedsHeaderString bool
}

type RuntimeTypeMeta struct {
	HTTPResponseType   string
	ServerRequestType  string
	ServerResponseType string
}

type RuntimeRouteMeta struct {
	Handlers     []intrinsics.HTTPHandlerSpec
	RouteStrings []string
}

func OrderedRuntimeFeatures() []RuntimeFeature {
	return []RuntimeFeature{
		RuntimeFeatureGoCore,
		RuntimeFeatureGoStdRuntime,
		RuntimeFeatureGoJSONWeb,
		RuntimeFeatureGoHTTP,
		RuntimeFeatureGoCrypto,
		RuntimeFeatureGoHTTPServe,
		RuntimeFeatureGoSession,
		RuntimeFeatureGoDB,
		RuntimeFeatureGoImportHMAC,
		RuntimeFeatureGoImportBcrypt,
		RuntimeFeatureGoImportSync,
		RuntimeFeatureLLVMStringGlobals,
		RuntimeFeatureLLVMRouteTable,
		RuntimeFeatureLLVMStringRuntime,
		RuntimeFeatureLLVMBuiltin,
		RuntimeFeatureLLVMAnyRuntime,
		RuntimeFeatureLLVMStdDecls,
		RuntimeFeatureLLVMParseInt,
		RuntimeFeatureLLVMParseFloat,
	}
}

func HasRuntimeFeature(surface RuntimeFeatureSurfaceMeta, feature RuntimeFeature) bool {
	for _, f := range surface.Ordered {
		if f == feature {
			return true
		}
	}
	return false
}

func CollectRuntimeFeatureSurfaceMeta(caps RuntimeCapabilityMeta) RuntimeFeatureSurfaceMeta {
	ordered := make([]RuntimeFeature, 0, len(OrderedRuntimeFeatures()))
	for _, feature := range OrderedRuntimeFeatures() {
		switch feature {
		case RuntimeFeatureGoCore,
			RuntimeFeatureGoStdRuntime,
			RuntimeFeatureGoJSONWeb,
			RuntimeFeatureGoHTTP,
			RuntimeFeatureGoCrypto,
			RuntimeFeatureLLVMStringGlobals,
			RuntimeFeatureLLVMRouteTable,
			RuntimeFeatureLLVMStringRuntime,
			RuntimeFeatureLLVMBuiltin,
			RuntimeFeatureLLVMAnyRuntime,
			RuntimeFeatureLLVMStdDecls:
			ordered = append(ordered, feature)
		case RuntimeFeatureGoHTTPServe:
			if caps.NeedsHTTPServeApp {
				ordered = append(ordered, feature)
			}
		case RuntimeFeatureGoSession:
			if caps.NeedsSession {
				ordered = append(ordered, feature)
			}
		case RuntimeFeatureGoDB:
			if caps.NeedsDB {
				ordered = append(ordered, feature)
			}
		case RuntimeFeatureGoImportHMAC:
			if caps.NeedsHMAC {
				ordered = append(ordered, feature)
			}
		case RuntimeFeatureGoImportBcrypt:
			if caps.NeedsBcrypt {
				ordered = append(ordered, feature)
			}
		case RuntimeFeatureGoImportSync:
			if caps.NeedsSession {
				ordered = append(ordered, feature)
			}
		case RuntimeFeatureLLVMParseInt:
			if caps.HasErrorType {
				ordered = append(ordered, feature)
			}
		case RuntimeFeatureLLVMParseFloat:
			if caps.HasErrorType {
				ordered = append(ordered, feature)
			}
		}
	}
	return RuntimeFeatureSurfaceMeta{Ordered: ordered}
}

func CollectProgramRuntimeMeta(p *mir.Program) ProgramRuntimeMeta {
	handlers := CollectHTTPHandlers(p)
	hasHTTPResponse := HasProgramTypeName(p, "HttpResponse")
	hasErrorType := HasProgramTypeName(p, "Error")
	hasHTTPHandlers := len(handlers) > 0
	needsSession := mir.ProgramUsesCall(p, intrinsics.IsSessionCall)
	needsDB := mir.ProgramUsesCall(p, intrinsics.IsDBCall) || needsSession
	needsBcrypt := mir.ProgramUsesCall(p, intrinsics.IsBcryptCall)
	needsJWT := mir.ProgramUsesCall(p, intrinsics.IsJWTCall)
	needsHMAC := mir.ProgramUsesCall(p, intrinsics.IsHMACCall) || needsJWT
	needsHTTPServeApp := mir.ProgramUsesCall(p, intrinsics.IsHTTPServeAppCall) || hasHTTPHandlers
	capabilities := RuntimeCapabilityMeta{
		HasErrorType:      hasErrorType,
		HasHTTPHandlers:   hasHTTPHandlers,
		HasHTTPResponse:   hasHTTPResponse,
		NeedsSession:      needsSession,
		NeedsDB:           needsDB,
		NeedsBcrypt:       needsBcrypt,
		NeedsJWT:          needsJWT,
		NeedsHMAC:         needsHMAC,
		NeedsHTTPServeApp: needsHTTPServeApp,
		NeedsHeaderString: hasHTTPResponse || hasHTTPHandlers,
	}
	return ProgramRuntimeMeta{
		Capabilities: capabilities,
		Types: RuntimeTypeMeta{
			HTTPResponseType:   ResolveProgramTypeName(p, "HttpResponse"),
			ServerRequestType:  ResolveProgramTypeName(p, "ServerRequest"),
			ServerResponseType: ResolveProgramTypeName(p, "ServerResponse"),
		},
		Routes: RuntimeRouteMeta{
			Handlers:     handlers,
			RouteStrings: RuntimeRouteStrings(handlers),
		},
		Features: CollectRuntimeFeatureSurfaceMeta(capabilities),
	}
}
