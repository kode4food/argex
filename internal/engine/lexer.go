package engine

import (
	"encoding/json/jsontext"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type (
	// tokenKind identifies a lexical token's category
	tokenKind int

	token struct {
		kind  tokenKind
		text  string
		value any
		pos   int
	}

	// scanner walks the source by byte offset and counts characters, so error
	// positions match what the author sees
	scanner struct {
		source     string
		byteOffset int
		charPos    int
		scratch    []byte
	}
)

const (
	kindEOF tokenKind = iota
	kindName
	kindLiteral

	// punctuation
	kindLParen
	kindRParen
	kindLBrack
	kindRBrack
	kindLBrace
	kindRBrace
	kindDot
	kindOptDot
	kindComma
	kindColon
	kindPipe
	kindSpread
	kindAssign

	// operators. kindIn never comes from the lexer, only from the parser
	kindIn
	kindNot
	kindAdd
	kindSub
	kindMul
	kindDiv
	kindMod
	kindLt
	kindGt
	kindEq
	kindNeq
	kindLte
	kindGte
	kindAnd
	kindOr
	kindCoalesce
)

// operator symbols, stored as an AST node's text so the compiler and static
// checker can switch on them without reaching back into the token layer
const (
	opAdd      = "+"
	opSub      = "-"
	opMul      = "*"
	opDiv      = "/"
	opMod      = "%"
	opNot      = "!"
	opEq       = "=="
	opNeq      = "!="
	opLt       = "<"
	opLte      = "<="
	opGt       = ">"
	opGte      = ">="
	opIn       = "in"
	opAnd      = "&&"
	opOr       = "||"
	opCoalesce = "??"
)

const (
	// pairLength is the length of every operator in pairKinds
	pairLength = 2

	// minUnescaped is the lowest byte JSON allows unescaped in a string
	minUnescaped = 0x20

	// bytesPerToken is a low estimate of source bytes per token, so the token
	// slice is sized once and rarely regrows
	bytesPerToken = 4
)

// operatorText maps an operator token back to its symbol, for storing on the
// AST node that the parser builds from it
var operatorText = map[tokenKind]string{
	kindIn:       opIn,
	kindNot:      opNot,
	kindAdd:      opAdd,
	kindSub:      opSub,
	kindMul:      opMul,
	kindDiv:      opDiv,
	kindMod:      opMod,
	kindLt:       opLt,
	kindGt:       opGt,
	kindEq:       opEq,
	kindNeq:      opNeq,
	kindLte:      opLte,
	kindGte:      opGte,
	kindAnd:      opAnd,
	kindOr:       opOr,
	kindCoalesce: opCoalesce,
}

var singleCharKinds = map[rune]tokenKind{
	'(': kindLParen,
	')': kindRParen,
	'[': kindLBrack,
	']': kindRBrack,
	'{': kindLBrace,
	'}': kindRBrace,
	'.': kindDot,
	',': kindComma,
	':': kindColon,
	'|': kindPipe,
	'+': kindAdd,
	'-': kindSub,
	'*': kindMul,
	'/': kindDiv,
	'%': kindMod,
	'!': kindNot,
	'<': kindLt,
	'>': kindGt,
	'=': kindAssign,
}

var punctuationText = map[tokenKind]string{
	kindEOF:     "end of input",
	kindName:    "name",
	kindLiteral: "literal",
	kindLParen:  "(",
	kindRParen:  ")",
	kindLBrack:  "[",
	kindRBrack:  "]",
	kindLBrace:  "{",
	kindRBrace:  "}",
	kindDot:     ".",
	kindOptDot:  "?.",
	kindComma:   ",",
	kindColon:   ":",
	kindPipe:    "|",
	kindSpread:  "...",
	kindAssign:  "=",
}

var pairKinds = map[string]tokenKind{
	"?.": kindOptDot,
	"??": kindCoalesce,
	"&&": kindAnd,
	"||": kindOr,
	"==": kindEq,
	"!=": kindNeq,
	"<=": kindLte,
	">=": kindGte,
}

// String renders a token kind for use in a syntax error message
func (k tokenKind) String() string {
	if text, ok := punctuationText[k]; ok {
		return text
	}
	return operatorText[k]
}

func lex(source string) ([]token, error) {
	s := &scanner{source: source}
	result := make([]token, 0, len(source)/bytesPerToken+1)
	s.skipIgnored()
	for s.byteOffset < len(s.source) {
		item, err := s.scan()
		if err != nil {
			return nil, err
		}
		result = append(result, item)
		s.skipIgnored()
	}
	return append(result, token{kind: kindEOF, pos: s.charPos}), nil
}

func (s *scanner) advance(size int) {
	end := s.byteOffset + size
	s.charPos += utf8.RuneCountInString(s.source[s.byteOffset:end])
	s.byteOffset = end
}

// skipIgnored advances past whitespace and comments to the next token
func (s *scanner) skipIgnored() {
	for s.byteOffset < len(s.source) {
		rest := s.source[s.byteOffset:]
		if strings.HasPrefix(rest, "//") {
			end := strings.IndexByte(rest, '\n')
			if end < 0 {
				end = len(rest)
			}
			s.advance(end)
			continue
		}
		r, size := utf8.DecodeRuneInString(rest)
		if !unicode.IsSpace(r) {
			return
		}
		s.advance(size)
	}
}

func (s *scanner) scan() (token, error) {
	start := s.charPos
	rest := s.source[s.byteOffset:]
	r, size := utf8.DecodeRuneInString(rest)
	if isNameStart(r) {
		n := size
		for n < len(rest) {
			next, width := utf8.DecodeRuneInString(rest[n:])
			if !isNamePart(next) {
				break
			}
			n += width
		}
		s.advance(n)
		return token{kind: kindName, text: rest[:n], pos: start}, nil
	}
	if r >= '0' && r <= '9' {
		return s.scanNumber()
	}
	if r == '"' {
		return s.scanString()
	}
	if strings.HasPrefix(rest, "...") {
		s.advance(len("..."))
		return token{kind: kindSpread, pos: start}, nil
	}
	if len(rest) >= pairLength {
		if kind, ok := pairKinds[rest[:pairLength]]; ok {
			s.advance(pairLength)
			return token{kind: kind, pos: start}, nil
		}
	}
	if kind, ok := singleCharKinds[r]; ok {
		s.advance(size)
		return token{kind: kind, pos: start}, nil
	}
	return token{}, syntax(start, "unexpected character")
}

func (s *scanner) scanNumber() (token, error) {
	start := s.charPos
	rest := s.source[s.byteOffset:]
	n := 0
	for n < len(rest) {
		c := rest[n]
		if strings.IndexByte("0123456789.eE", c) >= 0 {
			n++
			continue
		}
		if (c == '+' || c == '-') && n > 0 &&
			(rest[n-1] == 'e' || rest[n-1] == 'E') {
			n++
			continue
		}
		break
	}
	raw := rest[:n]
	s.advance(n)
	if !s.isNumber(raw) {
		return token{}, syntax(start, "invalid number")
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return token{}, syntax(start, "number out of range")
	}
	return token{kind: kindLiteral, value: value, pos: start}, nil
}

// scanString skips escaped bytes. UTF-8 continuation bytes are never a quote or
// backslash, so skipping one byte of a multibyte escape is safe
func (s *scanner) scanString() (token, error) {
	start := s.charPos
	rest := s.source[s.byteOffset:]
	for n := 1; n < len(rest); n++ {
		if rest[n] == '\\' {
			n++
			continue
		}
		if rest[n] != '"' {
			continue
		}
		value, err := s.unquote(rest[:n+1])
		s.advance(n + 1)
		if err != nil {
			return token{}, syntax(start, "invalid string")
		}
		return token{kind: kindLiteral, value: value, pos: start}, nil
	}
	s.advance(len(rest))
	return token{}, syntax(start, "unterminated string")
}

func (s *scanner) isNumber(text string) bool {
	var err error
	s.scratch, err = jsontext.AppendFormat(s.scratch[:0], text)
	return err == nil
}

// unquote decodes a JSON string literal. Without escapes or control bytes, a
// valid literal is its own text, so it is returned without copying
func (s *scanner) unquote(quoted string) (string, error) {
	body := quoted[1 : len(quoted)-1]
	if isPlain(body) && utf8.ValidString(body) {
		return body, nil
	}
	var err error
	s.scratch, err = jsontext.AppendUnquote(s.scratch[:0], quoted)
	if err != nil {
		return "", err
	}
	return string(s.scratch), nil
}

func isPlain(body string) bool {
	for i := range len(body) {
		if body[i] == '\\' || body[i] < minUnescaped {
			return false
		}
	}
	return true
}

func isNameStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isNamePart(r rune) bool {
	return isNameStart(r) || unicode.IsDigit(r)
}

func syntax(pos int, message string) error {
	return fmt.Errorf("%w at character %d: %s", ErrSyntax, pos, message)
}
