package sema

import (
	"fmt"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/source"
)

type scopeStack struct {
	frames []map[string]*varInfo
}

func newScopeStack() *scopeStack {
	return &scopeStack{frames: []map[string]*varInfo{}}
}

func (s *scopeStack) push() {
	s.frames = append(s.frames, map[string]*varInfo{})
}

func (s *scopeStack) pop() map[string]*varInfo {
	if len(s.frames) == 0 {
		return nil
	}
	scope := s.frames[len(s.frames)-1]
	s.frames = s.frames[:len(s.frames)-1]
	return scope
}

func (s *scopeStack) declare(name string, t ast.Type, isConst bool, span source.Span) error {
	scope := s.frames[len(s.frames)-1]
	if name == "_" {
		scope[name+fmt.Sprintf("#%d", len(scope))] = &varInfo{typ: t, used: true, declSpan: span, isConst: false}
		return nil
	}
	if _, exists := scope[name]; exists {
		return fmt.Errorf("duplicate variable '%s'", name)
	}
	scope[name] = &varInfo{typ: t, used: false, declSpan: span, isConst: isConst}
	return nil
}

func (s *scopeStack) resolve(name string, markUsed bool) (ast.Type, bool, bool) {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if v, ok := s.frames[i][name]; ok {
			if markUsed {
				v.used = true
			}
			return v.typ, v.isConst, true
		}
	}
	return ast.TypeInvalid, false, false
}

func (s *scopeStack) visibleNames() []string {
	names := map[string]bool{}
	for i := len(s.frames) - 1; i >= 0; i-- {
		for name := range s.frames[i] {
			if strings.HasPrefix(name, "_#") {
				continue
			}
			names[name] = true
		}
	}
	return mapKeys(names)
}

func (s *scopeStack) validateScopeUsage(scope map[string]*varInfo) error {
	for name, info := range scope {
		if strings.HasPrefix(name, "_#") {
			continue
		}
		if !info.used {
			return fmt.Errorf("unused variable '%s' (use '_' to ignore)", name)
		}
	}
	return nil
}
