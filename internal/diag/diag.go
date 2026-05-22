package diag

import (
	"errors"
	"fmt"
	"strings"

	"baziclang/internal/source"
)

type Error struct {
	Kind    string
	Message string
	Span    source.Span
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Span.IsZero() {
		return fmt.Sprintf("%s: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("%s at %d:%d: %s", e.Kind, e.Span.Start.Line, e.Span.Start.Col, e.Message)
}

func New(kind, message string, span source.Span) error {
	return &Error{Kind: kind, Message: message, Span: span}
}

func Extract(err error) (*Error, bool) {
	var out *Error
	if errors.As(err, &out) {
		return out, true
	}
	return nil, false
}

func RenderWithSource(err error, src, sourceName string) error {
	derr, ok := Extract(err)
	if !ok || derr.Span.IsZero() {
		return err
	}
	lines := strings.Split(src, "\n")
	line := derr.Span.Start.Line
	col := derr.Span.Start.Col
	if line <= 0 || line > len(lines) {
		return err
	}
	text := lines[line-1]
	caretCol := col
	if caretCol > len([]rune(text))+1 {
		caretCol = len([]rune(text)) + 1
	}
	spaces := strings.Repeat(" ", max(0, caretCol-1))
	underline := "^"
	if derr.Span.Start.File == derr.Span.End.File && derr.Span.Start.Line == derr.Span.End.Line {
		width := derr.Span.End.Col - derr.Span.Start.Col
		if width > 1 {
			underline = "^" + strings.Repeat("~", width-1)
		}
	}
	return fmt.Errorf("%s\n --> %s:%d:%d\n  |\n%3d | %s\n  | %s%s", derr.Error(), sourceName, line, col, line, text, spaces, underline)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
