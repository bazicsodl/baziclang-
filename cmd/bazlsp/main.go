package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"baziclang/internal/bazfmt"
	"baziclang/internal/diag"
	"baziclang/internal/intrinsics"
	"baziclang/internal/langsurface"
	"baziclang/internal/lexer"
	"baziclang/internal/parser"
	"baziclang/internal/sema"
)

type request struct {
	Jsonrpc string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type response struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *respError      `json:"error,omitempty"`
}

type respError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type docState struct {
	Text    string
	Symbols map[string]position
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type diagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"`
	Message  string   `json:"message"`
	Source   string   `json:"source"`
}

type codeAction struct {
	Title       string         `json:"title"`
	Kind        string         `json:"kind,omitempty"`
	Diagnostics []diagnostic   `json:"diagnostics,omitempty"`
	Edit        *workspaceEdit `json:"edit,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
}

type codeActionParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Range   lspRange `json:"range"`
	Context struct {
		Diagnostics []diagnostic `json:"diagnostics"`
	} `json:"context"`
}

type documentFormattingParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
}

type publishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

type completionItem struct {
	Label string `json:"label"`
	Kind  int    `json:"kind"`
}

type location struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

type textEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}

type workspaceEdit struct {
	Changes map[string][]textEdit `json:"changes"`
}

type docSymbol struct {
	Name           string   `json:"name"`
	Kind           int      `json:"kind"`
	Range          lspRange `json:"range"`
	SelectionRange lspRange `json:"selectionRange"`
}

type hover struct {
	Contents string `json:"contents"`
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	docs := map[string]*docState{}
	shutdown := false
	for {
		msg, err := readMessage(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			continue
		}
		var req request
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		switch req.Method {
		case "initialize":
			result := map[string]any{
				"capabilities": map[string]any{
					"textDocumentSync": 1,
					"completionProvider": map[string]any{
						"resolveProvider": false,
					},
					"definitionProvider":         true,
					"renameProvider":             true,
					"documentSymbolProvider":     true,
					"hoverProvider":              true,
					"codeActionProvider":         true,
					"documentFormattingProvider": true,
				},
			}
			writeResponse(req.ID, result)
		case "shutdown":
			shutdown = true
			writeResponse(req.ID, nil)
		case "exit":
			if shutdown {
				os.Exit(0)
			}
			os.Exit(1)
		case "textDocument/didOpen":
			var params struct {
				TextDocument struct {
					URI  string `json:"uri"`
					Text string `json:"text"`
				} `json:"textDocument"`
			}
			_ = json.Unmarshal(req.Params, &params)
			docs[params.TextDocument.URI] = &docState{
				Text:    params.TextDocument.Text,
				Symbols: indexSymbols(params.TextDocument.Text),
			}
			publishDiagnostics(params.TextDocument.URI, params.TextDocument.Text)
		case "textDocument/didChange":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				ContentChanges []struct {
					Text string `json:"text"`
				} `json:"contentChanges"`
			}
			_ = json.Unmarshal(req.Params, &params)
			if len(params.ContentChanges) == 0 {
				continue
			}
			text := params.ContentChanges[len(params.ContentChanges)-1].Text
			docs[params.TextDocument.URI] = &docState{
				Text:    text,
				Symbols: indexSymbols(text),
			}
			publishDiagnostics(params.TextDocument.URI, text)
		case "textDocument/didSave":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Text string `json:"text"`
			}
			_ = json.Unmarshal(req.Params, &params)
			state := docs[params.TextDocument.URI]
			if state != nil {
				publishDiagnostics(params.TextDocument.URI, state.Text)
			}
		case "textDocument/completion":
			writeResponse(req.ID, completionItemsForSurface())
		case "textDocument/definition":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position position `json:"position"`
			}
			_ = json.Unmarshal(req.Params, &params)
			state := docs[params.TextDocument.URI]
			if state == nil {
				writeResponse(req.ID, []location{})
				continue
			}
			word := wordAt(state.Text, params.Position)
			if word == "" {
				writeResponse(req.ID, []location{})
				continue
			}
			pos, ok := state.Symbols[word]
			if !ok {
				writeResponse(req.ID, []location{})
				continue
			}
			loc := location{
				URI: params.TextDocument.URI,
				Range: lspRange{
					Start: pos,
					End:   pos,
				},
			}
			writeResponse(req.ID, []location{loc})
		case "textDocument/rename":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position position `json:"position"`
				NewName  string   `json:"newName"`
			}
			_ = json.Unmarshal(req.Params, &params)
			state := docs[params.TextDocument.URI]
			if state == nil {
				writeResponse(req.ID, workspaceEdit{Changes: map[string][]textEdit{}})
				continue
			}
			word := wordAt(state.Text, params.Position)
			if word == "" {
				writeResponse(req.ID, workspaceEdit{Changes: map[string][]textEdit{}})
				continue
			}
			edits := findWordEdits(state.Text, word, params.NewName)
			writeResponse(req.ID, workspaceEdit{Changes: map[string][]textEdit{
				params.TextDocument.URI: edits,
			}})
		case "textDocument/documentSymbol":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			_ = json.Unmarshal(req.Params, &params)
			state := docs[params.TextDocument.URI]
			if state == nil {
				writeResponse(req.ID, []docSymbol{})
				continue
			}
			symbols := indexDocSymbols(state.Text)
			writeResponse(req.ID, symbols)
		case "textDocument/hover":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position position `json:"position"`
			}
			_ = json.Unmarshal(req.Params, &params)
			state := docs[params.TextDocument.URI]
			if state == nil {
				writeResponse(req.ID, hover{Contents: ""})
				continue
			}
			word := wordAt(state.Text, params.Position)
			if word == "" {
				writeResponse(req.ID, hover{Contents: ""})
				continue
			}
			msg := hoverFor(word)
			writeResponse(req.ID, hover{Contents: msg})
		case "textDocument/formatting":
			var params documentFormattingParams
			_ = json.Unmarshal(req.Params, &params)
			state := docs[params.TextDocument.URI]
			if state == nil {
				writeResponse(req.ID, []textEdit{})
				continue
			}
			edits := formatEdits(state.Text)
			writeResponse(req.ID, edits)
		case "textDocument/codeAction":
			var params codeActionParams
			_ = json.Unmarshal(req.Params, &params)
			state := docs[params.TextDocument.URI]
			if state == nil {
				writeResponse(req.ID, []codeAction{})
				continue
			}
			actions := codeActionsFor(params.TextDocument.URI, state.Text, params.Context.Diagnostics)
			writeResponse(req.ID, actions)
		default:
			if req.ID != nil {
				writeResponse(req.ID, nil)
			}
		}
	}
}

func readMessage(r *bufio.Reader) ([]byte, error) {
	var length int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, err
			}
			length = n
		}
	}
	if length == 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	buf := make([]byte, length)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func writeResponse(id *json.RawMessage, result any) {
	if id == nil {
		return
	}
	resp := response{
		Jsonrpc: "2.0",
		ID:      *id,
		Result:  result,
	}
	writeMessage(resp)
}

func writeMessage(v any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(os.Stdout, "Content-Length: %d\r\n\r\n", len(data))
	_, _ = os.Stdout.Write(data)
}

func publishDiagnostics(uri, text string) {
	diags := []diagnostic{}
	if err := checkText(uri, text); err != nil {
		rng := diagnosticRange(err)
		diags = append(diags, diagnostic{
			Range:    rng,
			Severity: 1,
			Message:  err.Error(),
			Source:   "bazic",
		})
	}
	params := publishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	}
	notify("textDocument/publishDiagnostics", params)
}

func notify(method string, params any) {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	writeMessage(msg)
}

func checkText(uri, text string) error {
	tokens, err := lexer.New(text).Tokenize()
	if err != nil {
		return err
	}
	prog, err := parser.New(tokens).ParseProgram()
	if err != nil {
		return err
	}
	if err := sema.New().Check(prog); err != nil {
		return err
	}
	_ = uri
	return nil
}

func diagnosticRange(err error) lspRange {
	derr, ok := diag.Extract(err)
	if !ok || derr.Span.IsZero() {
		return lspRange{}
	}
	start := position{
		Line:      max(0, derr.Span.Start.Line-1),
		Character: max(0, derr.Span.Start.Col-1),
	}
	end := position{
		Line:      max(0, derr.Span.End.Line-1),
		Character: max(0, derr.Span.End.Col-1),
	}
	if end.Line < start.Line || (end.Line == start.Line && end.Character < start.Character) {
		end = start
	}
	return lspRange{Start: start, End: end}
}

func wordAt(text string, pos position) string {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	if pos.Character < 0 || pos.Character > len(line) {
		return ""
	}
	start := pos.Character
	for start > 0 && isIdentChar(rune(line[start-1])) {
		start--
	}
	end := pos.Character
	for end < len(line) && isIdentChar(rune(line[end])) {
		end++
	}
	if start == end {
		return ""
	}
	return line[start:end]
}

func isIdentChar(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

func indexSymbols(text string) map[string]position {
	out := map[string]position{}
	add := func(name string, idx int) {
		if _, ok := out[name]; ok {
			return
		}
		line, col := indexToLineCol(text, idx)
		out[name] = position{Line: line, Character: col}
	}
	re := regexp.MustCompile(`\b(fn|struct|enum|interface|let)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	matches := re.FindAllStringSubmatchIndex(text, -1)
	for _, m := range matches {
		if len(m) < 6 {
			continue
		}
		name := text[m[4]:m[5]]
		add(name, m[4])
	}
	return out
}

func indexDocSymbols(text string) []docSymbol {
	re := regexp.MustCompile(`\b(fn|struct|enum|interface|let)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	matches := re.FindAllStringSubmatchIndex(text, -1)
	out := make([]docSymbol, 0, len(matches))
	for _, m := range matches {
		if len(m) < 6 {
			continue
		}
		kindWord := text[m[2]:m[3]]
		name := text[m[4]:m[5]]
		startLine, startCol := indexToLineCol(text, m[4])
		endLine, endCol := indexToLineCol(text, m[5])
		kind := 13
		switch kindWord {
		case "fn":
			kind = 12
		case "struct":
			kind = 23
		case "enum":
			kind = 10
		case "interface":
			kind = 11
		case "let":
			kind = 13
		}
		rng := lspRange{
			Start: position{Line: startLine, Character: startCol},
			End:   position{Line: endLine, Character: endCol},
		}
		out = append(out, docSymbol{
			Name:           name,
			Kind:           kind,
			Range:          rng,
			SelectionRange: rng,
		})
	}
	return out
}

func hoverFor(word string) string {
	if spec, ok := langsurface.LookupKeyword(word); ok {
		return spec.Hover
	}
	if spec, ok := intrinsics.LookupSurfaceFunction(word); ok {
		return spec.Hover()
	}
	return ""
}

func completionItemsForSurface() []completionItem {
	items := make([]completionItem, 0, len(langsurface.KeywordSpecs())+len(intrinsics.SurfaceFunctionSpecs()))
	for _, spec := range langsurface.KeywordSpecs() {
		items = append(items, completionItem{Label: spec.Name, Kind: spec.CompletionKind})
	}
	for _, spec := range intrinsics.SurfaceFunctionSpecs() {
		items = append(items, completionItem{Label: spec.Name, Kind: 3})
	}
	return items
}

func indexToLineCol(text string, idx int) (int, int) {
	line, col := 0, 0
	for i, r := range text {
		if i >= idx {
			break
		}
		if r == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}

func codeActionsFor(uri string, text string, diags []diagnostic) []codeAction {
	actions := []codeAction{}
	for _, d := range diags {
		actions = append(actions, quickFixesFor(uri, text, d)...)
	}
	if formatted := formatEdits(text); len(formatted) > 0 {
		actions = append(actions, codeAction{
			Title: "Bazic: Format document",
			Kind:  "source.format",
			Edit:  &workspaceEdit{Changes: map[string][]textEdit{uri: formatted}},
		})
	}
	return actions
}

func quickFixesFor(uri string, text string, d diagnostic) []codeAction {
	msg := d.Message
	line := d.Range.Start.Line
	actions := []codeAction{}
	if strings.Contains(msg, "expected ';'") {
		if edit, ok := insertAtLineEnd(text, line, ";"); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Insert ';'",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
				IsPreferred: true,
			})
		}
	}
	if strings.Contains(msg, "expected '}'") {
		if edit, ok := insertAtLineEnd(text, line, "}"); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Insert '}'",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
			})
		}
	}
	if strings.Contains(msg, "expected ')'") {
		if edit, ok := insertAtLineEnd(text, line, ")"); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Insert ')'",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
			})
		}
	}
	if strings.Contains(msg, "expected ']'") {
		if edit, ok := insertAtLineEnd(text, line, "]"); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Insert ']'",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
			})
		}
	}
	if strings.Contains(msg, "unterminated string literal") {
		if edit, ok := insertAtLineEnd(text, line, "\""); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Close string literal",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
			})
		}
	}
	if strings.Contains(msg, "unterminated block comment") {
		if edit, ok := insertAtLineEnd(text, line, "*/"); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Close block comment",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
			})
		}
	}
	if strings.Contains(msg, "expected '&' after '&'") {
		if edit, ok := doubleOperatorOnLine(text, line, "&"); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Replace '&' with '&&'",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
				IsPreferred: true,
			})
		}
	}
	if strings.Contains(msg, "expected '|' after '|'") {
		if edit, ok := doubleOperatorOnLine(text, line, "|"); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Replace '|' with '||'",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
				IsPreferred: true,
			})
		}
	}
	if strings.Contains(msg, "invalid escape") {
		if edit, ok := escapeBackslashesOnLine(text, line); ok {
			actions = append(actions, codeAction{
				Title:       "Bazic: Escape backslashes in string",
				Kind:        "quickfix",
				Diagnostics: []diagnostic{d},
				Edit:        &workspaceEdit{Changes: map[string][]textEdit{uri: {edit}}},
			})
		}
	}
	return actions
}

func insertAtLineEnd(text string, line int, insert string) (textEdit, bool) {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return textEdit{}, false
	}
	raw := lines[line]
	trimmed := strings.TrimSpace(raw)
	if strings.HasSuffix(trimmed, insert) {
		return textEdit{}, false
	}
	pos := position{Line: line, Character: len(raw)}
	return textEdit{
		Range:   lspRange{Start: pos, End: pos},
		NewText: insert,
	}, true
}

func escapeBackslashesOnLine(text string, line int) (textEdit, bool) {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return textEdit{}, false
	}
	raw := lines[line]
	first := strings.Index(raw, "\"")
	last := strings.LastIndex(raw, "\"")
	if first == -1 || last <= first {
		return textEdit{}, false
	}
	before := raw[:first+1]
	body := raw[first+1 : last]
	after := raw[last:]
	if !strings.Contains(body, "\\") {
		return textEdit{}, false
	}
	body = strings.ReplaceAll(body, "\\", "\\\\")
	updated := before + body + after
	return textEdit{
		Range: lspRange{
			Start: position{Line: line, Character: 0},
			End:   position{Line: line, Character: len(raw)},
		},
		NewText: updated,
	}, true
}

func doubleOperatorOnLine(text string, line int, op string) (textEdit, bool) {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return textEdit{}, false
	}
	raw := lines[line]
	idx := strings.Index(raw, op)
	for idx != -1 {
		if idx+1 < len(raw) && raw[idx+1:idx+2] == op {
			idx = strings.Index(raw[idx+2:], op)
			if idx != -1 {
				idx += 2
			}
			continue
		}
		updated := raw[:idx+1] + op + raw[idx+1:]
		return textEdit{
			Range: lspRange{
				Start: position{Line: line, Character: 0},
				End:   position{Line: line, Character: len(raw)},
			},
			NewText: updated,
		}, true
	}
	return textEdit{}, false
}

func formatEdits(text string) []textEdit {
	formatted, err := formatText(text)
	if err != nil || formatted == text {
		return []textEdit{}
	}
	end := documentEnd(text)
	return []textEdit{{
		Range: lspRange{
			Start: position{Line: 0, Character: 0},
			End:   end,
		},
		NewText: formatted,
	}}
}

func documentEnd(text string) position {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return position{Line: 0, Character: 0}
	}
	last := len(lines) - 1
	return position{Line: last, Character: len(lines[last])}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func formatText(text string) (string, error) {
	return bazfmt.Format(text)
}

func findWordEdits(text, word, newName string) []textEdit {
	if word == "" {
		return nil
	}
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	matches := pattern.FindAllStringIndex(text, -1)
	edits := make([]textEdit, 0, len(matches))
	for _, m := range matches {
		startLine, startCol := indexToLineCol(text, m[0])
		endLine, endCol := indexToLineCol(text, m[1])
		edits = append(edits, textEdit{
			Range: lspRange{
				Start: position{Line: startLine, Character: startCol},
				End:   position{Line: endLine, Character: endCol},
			},
			NewText: newName,
		})
	}
	return edits
}

// uriToPath intentionally omitted: current diagnostics are source-text based.
