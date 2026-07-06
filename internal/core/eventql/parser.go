package eventql

import (
	"fmt"
	"regexp"
)

// Compile parses and compiles an eventql query string. An empty string (or an
// empty selector "{}") compiles to a query that matches everything.
//
// Grammar:
//
//	query       := selector pipeline?
//	selector    := '{' (matcher (',' matcher)*)? '}'
//	matcher     := IDENT op STRING
//	op          := '=' | '!=' | '=~' | '!~'
//	pipeline    := stage*
//	stage       := lineFilter | labelFilter
//	lineFilter  := ('|=' | '|~' | '!=' | '!~') STRING
//	labelFilter := '|' IDENT op STRING
func Compile(input string) (*Query, error) {
	tokens, err := lex(input)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	q := &Query{}

	// An empty input matches everything.
	if p.cur().kind == tEOF {
		return q, nil
	}

	if err := p.parseSelector(q); err != nil {
		return nil, err
	}
	if err := p.parsePipeline(q); err != nil {
		return nil, err
	}
	if p.cur().kind != tEOF {
		return nil, fmt.Errorf("eventql: unexpected token %q at position %d", p.cur().val, p.cur().pos)
	}
	return q, nil
}

type parser struct {
	tokens []token
	i      int
}

func (p *parser) cur() token  { return p.tokens[p.i] }
func (p *parser) next() token { t := p.tokens[p.i]; p.i++; return t }

func (p *parser) parseSelector(q *Query) error {
	if p.cur().kind != tLBrace {
		return fmt.Errorf("eventql: expected '{' at position %d", p.cur().pos)
	}
	p.next() // consume '{'

	if p.cur().kind == tRBrace {
		p.next()
		return nil
	}

	for {
		if p.cur().kind != tIdent {
			return fmt.Errorf("eventql: expected label name at position %d", p.cur().pos)
		}
		name := p.next().val
		if _, ok := knownLabels[name]; !ok {
			return fmt.Errorf("eventql: unknown label %q (allowed: type, source, severity, org, user)", name)
		}
		op, err := p.parseOp()
		if err != nil {
			return err
		}
		if p.cur().kind != tString {
			return fmt.Errorf("eventql: expected quoted value at position %d", p.cur().pos)
		}
		value := p.next().val

		m := LabelMatcher{Name: name, Op: op, Value: value}
		if op == OpRe || op == OpNre {
			re, err := regexp.Compile(value)
			if err != nil {
				return fmt.Errorf("eventql: invalid regexp %q: %w", value, err)
			}
			m.re = re
		}
		q.Labels = append(q.Labels, m)

		switch p.cur().kind {
		case tComma:
			p.next()
			continue
		case tRBrace:
			p.next()
			return nil
		default:
			return fmt.Errorf("eventql: expected ',' or '}' at position %d", p.cur().pos)
		}
	}
}

func (p *parser) parsePipeline(q *Query) error {
	for {
		switch p.cur().kind {
		case tPipeEq:
			p.next()
			if err := p.appendLine(q, LineContains); err != nil {
				return err
			}
		case tPipeRe:
			p.next()
			if err := p.appendLine(q, LineRe); err != nil {
				return err
			}
		case tNeq:
			p.next()
			if err := p.appendLine(q, LineNotContains); err != nil {
				return err
			}
		case tNre:
			p.next()
			if err := p.appendLine(q, LineNre); err != nil {
				return err
			}
		case tPipe:
			p.next()
			if err := p.parseLabelFilter(q); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func (p *parser) appendLine(q *Query, op LineOp) error {
	if p.cur().kind != tString {
		return fmt.Errorf("eventql: expected quoted value for line filter at position %d", p.cur().pos)
	}
	value := p.next().val
	m := LineMatcher{Op: op, Value: value}
	if op == LineRe || op == LineNre {
		re, err := regexp.Compile(value)
		if err != nil {
			return fmt.Errorf("eventql: invalid regexp %q: %w", value, err)
		}
		m.re = re
	}
	q.Lines = append(q.Lines, m)
	return nil
}

func (p *parser) parseLabelFilter(q *Query) error {
	if p.cur().kind != tIdent {
		return fmt.Errorf("eventql: expected attribute name after '|' at position %d", p.cur().pos)
	}
	key := p.next().val
	op, err := p.parseOp()
	if err != nil {
		return err
	}
	if p.cur().kind != tString {
		return fmt.Errorf("eventql: expected quoted value at position %d", p.cur().pos)
	}
	value := p.next().val
	m := AttrMatcher{Key: key, Op: op, Value: value}
	if op == OpRe || op == OpNre {
		re, err := regexp.Compile(value)
		if err != nil {
			return fmt.Errorf("eventql: invalid regexp %q: %w", value, err)
		}
		m.re = re
	}
	q.Attrs = append(q.Attrs, m)
	return nil
}

func (p *parser) parseOp() (Op, error) {
	switch p.cur().kind {
	case tEq:
		p.next()
		return OpEq, nil
	case tNeq:
		p.next()
		return OpNeq, nil
	case tRe:
		p.next()
		return OpRe, nil
	case tNre:
		p.next()
		return OpNre, nil
	default:
		return 0, fmt.Errorf("eventql: expected operator (=, !=, =~, !~) at position %d", p.cur().pos)
	}
}
