package eventql

import (
	"fmt"
	"strings"
	"unicode"
)

type tokKind int

const (
	tEOF tokKind = iota
	tLBrace
	tRBrace
	tComma
	tIdent
	tString
	tEq     // =
	tNeq    // !=
	tRe     // =~
	tNre    // !~
	tPipe   // |
	tPipeEq // |=
	tPipeRe // |~
)

type token struct {
	kind tokKind
	val  string
	pos  int
}

// lex tokenizes an eventql query string.
func lex(input string) ([]token, error) {
	var tokens []token
	runes := []rune(input)
	i := 0
	n := len(runes)

	peek := func(off int) rune {
		if i+off < n {
			return runes[i+off]
		}
		return 0
	}

	for i < n {
		c := runes[i]

		switch {
		case unicode.IsSpace(c):
			i++
			continue
		case c == '{':
			tokens = append(tokens, token{tLBrace, "{", i})
			i++
		case c == '}':
			tokens = append(tokens, token{tRBrace, "}", i})
			i++
		case c == ',':
			tokens = append(tokens, token{tComma, ",", i})
			i++
		case c == '|':
			switch peek(1) {
			case '=':
				tokens = append(tokens, token{tPipeEq, "|=", i})
				i += 2
			case '~':
				tokens = append(tokens, token{tPipeRe, "|~", i})
				i += 2
			default:
				tokens = append(tokens, token{tPipe, "|", i})
				i++
			}
		case c == '=':
			if peek(1) == '~' {
				tokens = append(tokens, token{tRe, "=~", i})
				i += 2
			} else {
				tokens = append(tokens, token{tEq, "=", i})
				i++
			}
		case c == '!':
			switch peek(1) {
			case '=':
				tokens = append(tokens, token{tNeq, "!=", i})
				i += 2
			case '~':
				tokens = append(tokens, token{tNre, "!~", i})
				i += 2
			default:
				return nil, fmt.Errorf("eventql: unexpected character %q at position %d", string(c), i)
			}
		case c == '"':
			s, ni, err := lexString(runes, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{tString, s, i})
			i = ni
		case isIdentStart(c):
			start := i
			for i < n && isIdentChar(runes[i]) {
				i++
			}
			tokens = append(tokens, token{tIdent, string(runes[start:i]), start})
		default:
			return nil, fmt.Errorf("eventql: unexpected character %q at position %d", string(c), i)
		}
	}

	tokens = append(tokens, token{tEOF, "", n})
	return tokens, nil
}

// lexString reads a double-quoted string starting at runes[start] (== '"').
// It supports \" and \\ escapes. Returns the unquoted value and the next index.
func lexString(runes []rune, start int) (string, int, error) {
	var sb strings.Builder
	i := start + 1
	n := len(runes)
	for i < n {
		c := runes[i]
		if c == '\\' && i+1 < n {
			next := runes[i+1]
			switch next {
			case '"', '\\':
				sb.WriteRune(next)
				i += 2
				continue
			case 'n':
				sb.WriteRune('\n')
				i += 2
				continue
			case 't':
				sb.WriteRune('\t')
				i += 2
				continue
			}
		}
		if c == '"' {
			return sb.String(), i + 1, nil
		}
		sb.WriteRune(c)
		i++
	}
	return "", 0, fmt.Errorf("eventql: unterminated string starting at position %d", start)
}

func isIdentStart(c rune) bool {
	return unicode.IsLetter(c) || c == '_'
}

func isIdentChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '.' || c == '-' || c == ':'
}
