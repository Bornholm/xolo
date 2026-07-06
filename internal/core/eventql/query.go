// Package eventql implements a small, Loki/LogQL-inspired query language used to
// filter events. A query is composed of a label selector over the indexed event
// labels and an optional pipeline of attribute and line (message) filters:
//
//	{type="auth.login.failed", severity=~"warning|error"} | provider_id="op" |~ "timeout"
//
// The package is intentionally free of any business dependency: it exposes a
// compiled Query that callers can both (a) translate to SQL (for push-down) and
// (b) evaluate in memory via Query.Match against a plain Fields struct.
package eventql

import (
	"regexp"
	"strings"
)

// Op is a label/attribute matcher operator.
type Op int

const (
	OpEq  Op = iota // =
	OpNeq           // !=
	OpRe            // =~
	OpNre           // !~
)

// LineOp is a message (line) filter operator.
type LineOp int

const (
	LineContains    LineOp = iota // |=
	LineNotContains               // !=
	LineRe                        // |~
	LineNre                       // !~
)

// Indexed label names allowed in the selector. They map to promoted columns on
// the event table.
const (
	LabelType     = "type"
	LabelSource   = "source"
	LabelSeverity = "severity"
	LabelOrg      = "org"
	LabelUser     = "user"
)

var knownLabels = map[string]struct{}{
	LabelType:     {},
	LabelSource:   {},
	LabelSeverity: {},
	LabelOrg:      {},
	LabelUser:     {},
}

// LabelMatcher matches an indexed label.
type LabelMatcher struct {
	Name  string
	Op    Op
	Value string
	re    *regexp.Regexp
}

func (m LabelMatcher) match(v string) bool { return matchOp(m.Op, m.re, m.Value, v) }

// AttrMatcher matches a free-form event attribute (key/value).
type AttrMatcher struct {
	Key   string
	Op    Op
	Value string
	re    *regexp.Regexp
}

func (m AttrMatcher) match(v string) bool { return matchOp(m.Op, m.re, m.Value, v) }

// LineMatcher matches against the event message.
type LineMatcher struct {
	Op    LineOp
	Value string
	re    *regexp.Regexp
}

func (m LineMatcher) match(msg string) bool {
	switch m.Op {
	case LineContains:
		return strings.Contains(msg, m.Value)
	case LineNotContains:
		return !strings.Contains(msg, m.Value)
	case LineRe:
		return m.re.MatchString(msg)
	case LineNre:
		return !m.re.MatchString(msg)
	}
	return false
}

// Query is a compiled event query.
type Query struct {
	Labels []LabelMatcher
	Attrs  []AttrMatcher
	Lines  []LineMatcher
}

// Fields is the plain projection of an event evaluated by Match.
type Fields struct {
	Type       string
	Source     string
	Severity   string
	Org        string
	User       string
	Message    string
	Attributes map[string]string
}

// Match reports whether the given fields satisfy every matcher in the query.
// A query with no matcher matches everything.
func (q *Query) Match(f Fields) bool {
	for _, m := range q.Labels {
		if !m.match(f.labelValue(m.Name)) {
			return false
		}
	}
	for _, m := range q.Attrs {
		if !m.match(f.Attributes[m.Key]) {
			return false
		}
	}
	for _, m := range q.Lines {
		if !m.match(f.Message) {
			return false
		}
	}
	return true
}

func (f Fields) labelValue(name string) string {
	switch name {
	case LabelType:
		return f.Type
	case LabelSource:
		return f.Source
	case LabelSeverity:
		return f.Severity
	case LabelOrg:
		return f.Org
	case LabelUser:
		return f.User
	}
	return ""
}

// HasRegex reports whether the query contains at least one regex matcher, which
// callers may not be able to push down to SQL.
func (q *Query) HasRegex() bool {
	for _, m := range q.Labels {
		if m.Op == OpRe || m.Op == OpNre {
			return true
		}
	}
	for _, m := range q.Attrs {
		if m.Op == OpRe || m.Op == OpNre {
			return true
		}
	}
	for _, m := range q.Lines {
		if m.Op == LineRe || m.Op == LineNre {
			return true
		}
	}
	return false
}

// IsEmpty reports whether the query has no matcher at all (matches everything).
func (q *Query) IsEmpty() bool {
	return len(q.Labels) == 0 && len(q.Attrs) == 0 && len(q.Lines) == 0
}

func matchOp(op Op, re *regexp.Regexp, pattern, v string) bool {
	switch op {
	case OpEq:
		return v == pattern
	case OpNeq:
		return v != pattern
	case OpRe:
		return re.MatchString(v)
	case OpNre:
		return !re.MatchString(v)
	}
	return false
}
