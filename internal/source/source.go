package source

type Position struct {
	File   string
	Offset int
	Line   int
	Col    int
}

type Span struct {
	Start Position
	End   Position
}

func Point(offset, line, col int) Span {
	pos := Position{Offset: offset, Line: line, Col: col}
	return Span{Start: pos, End: pos}
}

func Range(startOffset, startLine, startCol, endOffset, endLine, endCol int) Span {
	return Span{
		Start: Position{Offset: startOffset, Line: startLine, Col: startCol},
		End:   Position{Offset: endOffset, Line: endLine, Col: endCol},
	}
}

func Join(start, end Span) Span {
	if start.IsZero() {
		return end
	}
	if end.IsZero() {
		return start
	}
	return Span{Start: start.Start, End: end.End}
}

func (s Span) IsZero() bool {
	return s.Start == (Position{}) && s.End == (Position{})
}

func (s Span) WithFile(file string) Span {
	if s.IsZero() || file == "" {
		return s
	}
	out := s
	out.Start.File = file
	out.End.File = file
	return out
}
