package codegen

import (
	"testing"

	"baziclang/internal/ast"
)

func FuzzParseHttpHandlerName(f *testing.F) {
	f.Add("get_root")
	f.Add("post_users_p_id")
	f.Add("patch_posts_p_postId_comments")
	f.Fuzz(func(t *testing.T, name string) {
		fn := &ast.FuncDecl{
			Name: name,
			Params: []ast.Param{{Name: "req", Type: ast.Type("ServerRequest")}},
			ReturnType: ast.Type("ServerResponse"),
		}
		_, _ = parseHttpHandler(fn)
	})
}
