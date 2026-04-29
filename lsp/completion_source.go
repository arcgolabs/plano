package lsp

import (
	"go/token"
	"strings"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/internal/lexer"
)

func inferFormKindFromSource(file *token.File, src []byte, offset int) string {
	if file == nil {
		return ""
	}

	scan := newSourceFormScan(file, offset)
	scanner := lexer.New(file, src)
	for tok := scanner.Next(); scan.accept(tok); tok = scanner.Next() {
		scan.advance(tok)
	}
	return scan.currentFormKind()
}

type sourceFormScan struct {
	file       *token.File
	target     token.Pos
	line       int
	lineTokens []lexer.Token
	formStack  *list.List[string]
}

func newSourceFormScan(file *token.File, offset int) sourceFormScan {
	return sourceFormScan{
		file:      file,
		target:    file.Pos(offset),
		formStack: list.NewList[string](),
	}
}

func (s *sourceFormScan) accept(tok lexer.Token) bool {
	return tok.Kind != lexer.EOF && tok.Pos <= s.target
}

func (s *sourceFormScan) advance(tok lexer.Token) {
	s.resetLine(tok)
	if s.handleBrace(tok) {
		return
	}
	s.lineTokens = append(s.lineTokens, tok)
}

func (s *sourceFormScan) resetLine(tok lexer.Token) {
	line := s.file.Position(tok.Pos).Line
	if line == s.line {
		return
	}
	s.lineTokens = s.lineTokens[:0]
	s.line = line
}

func (s *sourceFormScan) handleBrace(tok lexer.Token) bool {
	if tok.Kind == lexer.LBrace {
		s.formStack.Add(inferFormKindFromHead(s.lineTokens))
		s.lineTokens = s.lineTokens[:0]
		return true
	}
	if tok.Kind == lexer.RBrace {
		s.popForm()
		s.lineTokens = s.lineTokens[:0]
		return true
	}
	return false
}

func (s *sourceFormScan) popForm() {
	if s.formStack.Len() == 0 {
		return
	}
	s.formStack.RemoveAt(s.formStack.Len() - 1)
}

func (s *sourceFormScan) currentFormKind() string {
	for index := s.formStack.Len() - 1; index >= 0; index-- {
		formKind, _ := s.formStack.Get(index)
		if formKind != "" {
			return formKind
		}
	}
	return ""
}

func inferFormKindFromHead(tokens []lexer.Token) string {
	parts, ok := formHeadParts(tokens)
	if !ok {
		return ""
	}

	head, consumed, ok := parseQualifiedIdentParts(parts)
	if !ok || head == "" {
		return ""
	}
	if isBareQualifiedHead(parts, consumed) || isLabeledQualifiedHead(tokens, parts, consumed) {
		return head
	}
	return ""
}

func formHeadParts(tokens []lexer.Token) ([]string, bool) {
	if len(tokens) == 0 || tokens[0].Kind != lexer.Ident {
		return nil, false
	}

	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token.Kind == lexer.Ident {
			parts = append(parts, token.Text)
			continue
		}
		if token.Kind == lexer.Dot {
			parts = append(parts, ".")
			continue
		}
		return nil, false
	}
	return parts, true
}

func isBareQualifiedHead(parts []string, consumed int) bool {
	return consumed == len(parts)
}

func isLabeledQualifiedHead(tokens []lexer.Token, parts []string, consumed int) bool {
	return consumed+1 == len(parts) && tokens[len(tokens)-1].Kind == lexer.Ident
}

func parseQualifiedIdentParts(parts []string) (string, int, bool) {
	if len(parts) == 0 || parts[0] == "." {
		return "", 0, false
	}

	segments := []string{parts[0]}
	index := 1
	for index+1 < len(parts) && parts[index] == "." && parts[index+1] != "." {
		segments = append(segments, parts[index+1])
		index += 2
	}
	return strings.Join(segments, "."), index, true
}
