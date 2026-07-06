package eventql

import "testing"

func TestCompileErrors(t *testing.T) {
	cases := []string{
		`{`,
		`{type}`,
		`{type=}`,
		`{type="a"`,
		`{unknown="a"}`,
		`{type=~"("}`,       // invalid regexp
		`{type="a"} | `,     // missing attr filter
		`{type="a"} |= `,    // missing line value
		`{type="a"} foo`,    // trailing garbage
		`{type="a"} |~ "("`, // invalid line regexp
	}
	for _, in := range cases {
		if _, err := Compile(in); err == nil {
			t.Errorf("expected error for %q, got nil", in)
		}
	}
}

func TestCompileEmpty(t *testing.T) {
	for _, in := range []string{"", "  ", "{}"} {
		q, err := Compile(in)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", in, err)
		}
		if !q.IsEmpty() {
			t.Errorf("expected empty query for %q", in)
		}
		if !q.Match(Fields{Type: "anything"}) {
			t.Errorf("empty query should match everything (%q)", in)
		}
	}
}

func TestMatch(t *testing.T) {
	base := Fields{
		Type:     "auth.login.failed",
		Source:   "platform",
		Severity: "warning",
		Org:      "org1",
		User:     "u1",
		Message:  "login failed: timeout contacting idp",
		Attributes: map[string]string{
			"provider_id": "op",
			"attempts":    "3",
		},
	}

	cases := []struct {
		query string
		want  bool
	}{
		{`{type="auth.login.failed"}`, true},
		{`{type="auth.login.ok"}`, false},
		{`{type!="auth.login.ok"}`, true},
		{`{severity=~"warning|error"}`, true},
		{`{severity=~"error"}`, false},
		{`{severity!~"error"}`, true},
		{`{type="auth.login.failed", user="u1"}`, true},
		{`{type="auth.login.failed", user="u2"}`, false},
		{`{source="platform"} | provider_id="op"`, true},
		{`{source="platform"} | provider_id="other"`, false},
		{`{source="platform"} | provider_id=~"o.*"`, true},
		{`{source="platform"} | missing="x"`, false}, // missing attr → "" != "x"
		{`{source="platform"} | missing!="x"`, true},
		{`{} |= "timeout"`, true},
		{`{} |= "success"`, false},
		{`{} != "success"`, true},
		{`{} |~ "time.ut"`, true},
		{`{} !~ "nope"`, true},
		{`{type="auth.login.failed"} |= "timeout" | attempts="3"`, true},
		{`{type="auth.login.failed"} |= "timeout" | attempts="5"`, false},
	}

	for _, tc := range cases {
		q, err := Compile(tc.query)
		if err != nil {
			t.Fatalf("compile %q: %v", tc.query, err)
		}
		if got := q.Match(base); got != tc.want {
			t.Errorf("Match(%q) = %v, want %v", tc.query, got, tc.want)
		}
	}
}

func TestHasRegex(t *testing.T) {
	cases := map[string]bool{
		`{type="a"}`:              false,
		`{type=~"a"}`:             true,
		`{type="a"} | k="v"`:      false,
		`{type="a"} | k=~"v"`:     true,
		`{type="a"} |= "v"`:       false,
		`{type="a"} |~ "v"`:       true,
		`{type="a"} != "v"`:       false,
	}
	for in, want := range cases {
		q, err := Compile(in)
		if err != nil {
			t.Fatalf("compile %q: %v", in, err)
		}
		if got := q.HasRegex(); got != want {
			t.Errorf("HasRegex(%q) = %v, want %v", in, got, want)
		}
	}
}
