package backendmeta

import "baziclang/internal/intrinsics"

type goRuntimeImportRule struct {
	feature     RuntimeFeature
	target      string
	targetNot   string
	importValue string
}

type goRuntimeHelperRule struct {
	feature RuntimeFeature
	section GoRuntimeHelperSection
}

type goRuntimeHelperSnippetRule struct {
	section GoRuntimeHelperSection
	build   func(meta ProgramRuntimeMeta, target string) string
}

type llvmPreludeSectionRule struct {
	feature RuntimeFeature
	section intrinsics.LLVMRuntimePreludeSection
}

type llvmBuiltinSectionRule struct {
	feature RuntimeFeature
	section intrinsics.LLVMBuiltinRuntimeSection
}

var baseGoRuntimeImports = []string{
	"\"bufio\"",
	"\"bytes\"",
	"\"crypto/rand\"",
	"\"crypto/sha256\"",
	"\"crypto/tls\"",
	"\"crypto/x509\"",
	"\"encoding/base64\"",
	"\"encoding/hex\"",
	"\"encoding/json\"",
	"\"fmt\"",
	"\"io\"",
	"\"math\"",
	"\"net\"",
	"\"net/http\"",
	"\"net/url\"",
	"\"os\"",
	"\"os/exec\"",
	"\"path/filepath\"",
	"\"runtime\"",
	"\"strconv\"",
	"\"strings\"",
	"\"sync\"",
	"\"time\"",
	"\"unicode/utf8\"",
}

var goRuntimeImportRules = []goRuntimeImportRule{
	{feature: RuntimeFeatureGoImportHMAC, importValue: "\"crypto/hmac\""},
	{feature: RuntimeFeatureGoImportBcrypt, importValue: "\"golang.org/x/crypto/bcrypt\""},
	{target: "wasm", importValue: "\"syscall/js\""},
	{feature: RuntimeFeatureGoDB, importValue: "\"database/sql\""},
	{feature: RuntimeFeatureGoDB, targetNot: "wasm", importValue: "_ \"github.com/go-sql-driver/mysql\""},
	{feature: RuntimeFeatureGoDB, targetNot: "wasm", importValue: "_ \"github.com/lib/pq\""},
	{feature: RuntimeFeatureGoDB, targetNot: "wasm", importValue: "_ \"modernc.org/sqlite\""},
}

var goRuntimeHelperRules = []goRuntimeHelperRule{
	{feature: RuntimeFeatureGoCore, section: GoRuntimeHelperCore},
	{feature: RuntimeFeatureGoStdRuntime, section: GoRuntimeHelperStdRuntime},
	{feature: RuntimeFeatureGoJSONWeb, section: GoRuntimeHelperJSONWeb},
	{feature: RuntimeFeatureGoHTTP, section: GoRuntimeHelperHTTP},
	{feature: RuntimeFeatureGoCrypto, section: GoRuntimeHelperCrypto},
	{feature: RuntimeFeatureGoHTTPServe, section: GoRuntimeHelperHTTPServe},
	{feature: RuntimeFeatureGoSession, section: GoRuntimeHelperSession},
	{feature: RuntimeFeatureGoDB, section: GoRuntimeHelperDB},
}

var goRuntimeHelperSnippetRules = []goRuntimeHelperSnippetRule{
	{
		section: GoRuntimeHelperCore,
		build: func(meta ProgramRuntimeMeta, target string) string {
			return intrinsics.GoHelperCode()
		},
	},
	{
		section: GoRuntimeHelperStdRuntime,
		build: func(meta ProgramRuntimeMeta, target string) string {
			return intrinsics.GoStdRuntimeHelperCode()
		},
	},
	{
		section: GoRuntimeHelperJSONWeb,
		build: func(meta ProgramRuntimeMeta, target string) string {
			return intrinsics.GoStdJSONWebHelperCode(target)
		},
	},
	{
		section: GoRuntimeHelperHTTP,
		build: func(meta ProgramRuntimeMeta, target string) string {
			return intrinsics.GoStdHTTPHelperCode(meta.Types.HTTPResponseType, meta.Capabilities.HasHTTPResponse, meta.Capabilities.NeedsHeaderString)
		},
	},
	{
		section: GoRuntimeHelperCrypto,
		build: func(meta ProgramRuntimeMeta, target string) string {
			return intrinsics.GoStdCryptoSystemHelperCode(meta.Capabilities.NeedsHMAC, meta.Capabilities.NeedsJWT, meta.Capabilities.NeedsBcrypt)
		},
	},
	{
		section: GoRuntimeHelperHTTPServe,
		build: func(meta ProgramRuntimeMeta, target string) string {
			return intrinsics.GoHTTPServeAppHelperCode(meta.Routes.Handlers, meta.Types.ServerRequestType, meta.Types.ServerResponseType)
		},
	},
	{
		section: GoRuntimeHelperSession,
		build: func(meta ProgramRuntimeMeta, target string) string {
			return intrinsics.GoStdSessionHelperCode()
		},
	},
	{
		section: GoRuntimeHelperDB,
		build: func(meta ProgramRuntimeMeta, target string) string {
			return intrinsics.GoStdDBHelperCode()
		},
	},
}

var llvmPreludeSectionRules = []llvmPreludeSectionRule{
	{feature: RuntimeFeatureLLVMStringGlobals, section: intrinsics.LLVMRuntimePreludeStringGlobals},
	{feature: RuntimeFeatureLLVMRouteTable, section: intrinsics.LLVMRuntimePreludeRouteTable},
	{feature: RuntimeFeatureLLVMStringRuntime, section: intrinsics.LLVMRuntimePreludeStringRuntime},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMRuntimePreludeBuiltin},
	{feature: RuntimeFeatureLLVMAnyRuntime, section: intrinsics.LLVMRuntimePreludeAnyRuntime},
	{feature: RuntimeFeatureLLVMStdDecls, section: intrinsics.LLVMRuntimePreludeStdDecls},
}

var llvmBuiltinSectionRules = []llvmBuiltinSectionRule{
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeContains},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeStartsWith},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeEndsWith},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeToUpper},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeToLower},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeTrimSpace},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeRepeat},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeReplace},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeIntToStr},
	{feature: RuntimeFeatureLLVMBuiltin, section: intrinsics.LLVMBuiltinRuntimeFloatToStr},
	{feature: RuntimeFeatureLLVMParseInt, section: intrinsics.LLVMBuiltinRuntimeParseInt},
	{feature: RuntimeFeatureLLVMParseFloat, section: intrinsics.LLVMBuiltinRuntimeParseFloat},
}

func collectGoRuntimeImports(features RuntimeFeatureSurfaceMeta, target string) []string {
	out := append([]string(nil), baseGoRuntimeImports...)
	for _, rule := range goRuntimeImportRules {
		if rule.feature != "" && !HasRuntimeFeature(features, rule.feature) {
			continue
		}
		if rule.target != "" && rule.target != target {
			continue
		}
		if rule.targetNot != "" && rule.targetNot == target {
			continue
		}
		out = append(out, rule.importValue)
	}
	return out
}

func collectGoRuntimeHelperSections(features RuntimeFeatureSurfaceMeta) []GoRuntimeHelperSection {
	sections := make([]GoRuntimeHelperSection, 0, len(goRuntimeHelperRules))
	for _, rule := range goRuntimeHelperRules {
		if HasRuntimeFeature(features, rule.feature) {
			sections = append(sections, rule.section)
		}
	}
	return sections
}

func collectGoRuntimeHelperSnippets(meta ProgramRuntimeMeta, target string, sections []GoRuntimeHelperSection) []string {
	snippets := make([]string, 0, len(sections))
	for _, section := range sections {
		for _, rule := range goRuntimeHelperSnippetRules {
			if rule.section == section {
				snippets = append(snippets, rule.build(meta, target))
				break
			}
		}
	}
	return snippets
}

func collectLLVMRuntimePreludeSections(features RuntimeFeatureSurfaceMeta) []intrinsics.LLVMRuntimePreludeSection {
	sections := make([]intrinsics.LLVMRuntimePreludeSection, 0, len(llvmPreludeSectionRules))
	for _, rule := range llvmPreludeSectionRules {
		if HasRuntimeFeature(features, rule.feature) {
			sections = append(sections, rule.section)
		}
	}
	return sections
}

func collectLLVMBuiltinRuntimeSections(features RuntimeFeatureSurfaceMeta) []intrinsics.LLVMBuiltinRuntimeSection {
	sections := make([]intrinsics.LLVMBuiltinRuntimeSection, 0, len(llvmBuiltinSectionRules))
	for _, rule := range llvmBuiltinSectionRules {
		if HasRuntimeFeature(features, rule.feature) {
			sections = append(sections, rule.section)
		}
	}
	return sections
}
