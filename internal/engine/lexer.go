package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type token struct {
	kind  string
	text  string
	value any
	pos   int
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
	return append(result, token{kind: "eof", pos: len(src)}), nil
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
			kind: "name",
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
		return token{kind: "...", pos: start}, pos + 3, nil
	}
	if pos+1 < len(src) {
		pair := string(src[pos : pos+2])
		switch pair {
		case "?.", "??", "&&", "||", "==", "!=", "<=", ">=":
			return token{kind: pair, pos: start}, pos + 2, nil
		}
	}
	if strings.ContainsRune("()[]{}.,:|+-*/%!<>", r) {
		return token{kind: string(r), pos: start}, pos + 1, nil
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
	return token{kind: "literal", value: value, pos: start}, pos, nil
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
		return token{kind: "literal", value: value, pos: start}, pos + 1, nil
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
