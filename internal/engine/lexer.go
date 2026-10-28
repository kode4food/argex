package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
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
	src := []rune(source)
	var result []token
	pos := skipIgnored(src, 0)
	for pos < len(src) {
		item, next, err := scan(src, pos)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
		pos = skipIgnored(src, next)
	}
	return append(result, token{kind: kindEOF, pos: len(src)}), nil
}

// skipIgnored returns the next token's position, past whitespace and comments
func skipIgnored(src []rune, pos int) int {
	for pos < len(src) {
		switch {
		case unicode.IsSpace(src[pos]):
			pos++

		case src[pos] == '/' && pos+1 < len(src) && src[pos+1] == '/':
			for pos < len(src) && src[pos] != '\n' {
				pos++
			}

		default:
			return pos
		}
	}
	return pos
}

func scan(src []rune, pos int) (token, int, error) {
	start := pos
	r := src[pos]
	if isNameStart(r) {
		pos++
		for pos < len(src) && isNamePart(src[pos]) {
			pos++
		}
		return token{
			kind: kindName,
			text: string(src[start:pos]),
			pos:  start,
		}, pos, nil
	}
	if r >= '0' && r <= '9' {
		return scanNumber(src, start)
	}
	if r == '"' {
		return scanString(src, start)
	}
	if pos+2 < len(src) && src[pos] == '.' && src[pos+1] == '.' &&
		src[pos+2] == '.' {
		return token{kind: kindSpread, pos: start}, pos + 3, nil
	}
	if pos+1 < len(src) {
		pair := string(src[pos : pos+2])
		if kind, ok := pairKinds[pair]; ok {
			return token{kind: kind, pos: start}, pos + 2, nil
		}
	}
	if kind, ok := singleCharKinds[r]; ok {
		return token{kind: kind, pos: start}, pos + 1, nil
	}
	return token{}, pos, syntax(start, "unexpected character")
}

func scanNumber(src []rune, start int) (token, int, error) {
	pos := start
	for pos < len(src) {
		r := src[pos]
		if strings.ContainsRune("0123456789.eE", r) {
			pos++
			continue
		}
		if (r == '+' || r == '-') && pos > start &&
			(src[pos-1] == 'e' || src[pos-1] == 'E') {
			pos++
			continue
		}
		break
	}
	raw := string(src[start:pos])
	if !json.Valid([]byte(raw)) {
		return token{}, pos, syntax(start, "invalid number")
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return token{}, pos, syntax(start, "number out of range")
	}
	return token{kind: kindLiteral, value: value, pos: start}, pos, nil
}

func scanString(src []rune, start int) (token, int, error) {
	for pos := start + 1; pos < len(src); pos++ {
		if src[pos] == '\\' {
			pos++
			continue
		}
		if src[pos] != '"' {
			continue
		}
		var value string
		err := json.Unmarshal([]byte(string(src[start:pos+1])), &value)
		if err != nil {
			return token{}, pos, syntax(start, "invalid string")
		}
		return token{kind: kindLiteral, value: value, pos: start}, pos + 1, nil
	}
	return token{}, len(src), syntax(start, "unterminated string")
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
