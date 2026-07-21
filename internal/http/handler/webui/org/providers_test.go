package org

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func makeRequest(fields map[string]string) *http.Request {
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	r, _ := http.NewRequest("POST", "/", nil)
	r.Form = form
	return r
}

func TestParseDurationField_Empty(t *testing.T) {
	r := makeRequest(map[string]string{"val": "", "unit": "s"})
	d, err := parseDurationField(r, "val", "unit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseDurationField_Seconds(t *testing.T) {
	r := makeRequest(map[string]string{"val": "5", "unit": "s"})
	d, err := parseDurationField(r, "val", "unit")
	if err != nil {
		t.Fatal(err)
	}
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
}

func TestParseDurationField_Milliseconds(t *testing.T) {
	r := makeRequest(map[string]string{"val": "500", "unit": "ms"})
	d, err := parseDurationField(r, "val", "unit")
	if err != nil {
		t.Fatal(err)
	}
	if d != 500*time.Millisecond {
		t.Errorf("expected 500ms, got %v", d)
	}
}

func TestParseDurationField_Minutes(t *testing.T) {
	r := makeRequest(map[string]string{"val": "2", "unit": "min"})
	d, err := parseDurationField(r, "val", "unit")
	if err != nil {
		t.Fatal(err)
	}
	if d != 2*time.Minute {
		t.Errorf("expected 2min, got %v", d)
	}
}

func TestParseDurationField_InvalidValue(t *testing.T) {
	r := makeRequest(map[string]string{"val": "abc", "unit": "s"})
	_, err := parseDurationField(r, "val", "unit")
	if err == nil {
		t.Error("expected error for non-numeric value")
	}
}

func TestParseDurationField_ZeroValue(t *testing.T) {
	r := makeRequest(map[string]string{"val": "0", "unit": "s"})
	_, err := parseDurationField(r, "val", "unit")
	if err == nil {
		t.Error("expected error for zero value")
	}
}

func TestParseDurationField_NegativeValue(t *testing.T) {
	r := makeRequest(map[string]string{"val": "-1", "unit": "s"})
	_, err := parseDurationField(r, "val", "unit")
	if err == nil {
		t.Error("expected error for negative value")
	}
}

func TestCoerceExtraBodyValue(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"true", true},
		{"TRUE", true},
		{"false", false},
		{"  true  ", true},
		{"40", int64(40)},
		{"-3", int64(-3)},
		{"0.7", float64(0.7)},
		{"auto", "auto"},
		{"", ""},
		{"12abc", "12abc"},
	}
	for _, c := range cases {
		got := coerceExtraBodyValue(c.in)
		if got != c.want {
			t.Errorf("coerceExtraBodyValue(%q) = %#v (%T), want %#v (%T)", c.in, got, got, c.want, c.want)
		}
	}
}

func TestParseExtraBodyFromForm(t *testing.T) {
	t.Run("no rows yields nil", func(t *testing.T) {
		r := makeRequest(map[string]string{"extra_body_count": "0"})
		m, err := parseExtraBodyFromForm(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m != nil {
			t.Errorf("expected nil map, got %v", m)
		}
	})

	t.Run("typed rows", func(t *testing.T) {
		r := makeRequest(map[string]string{
			"extra_body_count":     "3",
			"extra_body_k0_key":    "reasoning_split",
			"extra_body_k0_value":  "true",
			"extra_body_k1_key":    "top_k",
			"extra_body_k1_value":  "40",
			"extra_body_k2_key":    "mode",
			"extra_body_k2_value":  "auto",
		})
		m, err := parseExtraBodyFromForm(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["reasoning_split"] != true {
			t.Errorf("reasoning_split = %#v, want true", m["reasoning_split"])
		}
		if m["top_k"] != int64(40) {
			t.Errorf("top_k = %#v, want int64(40)", m["top_k"])
		}
		if m["mode"] != "auto" {
			t.Errorf("mode = %#v, want \"auto\"", m["mode"])
		}
	})

	t.Run("empty key rows are skipped", func(t *testing.T) {
		r := makeRequest(map[string]string{
			"extra_body_count":    "2",
			"extra_body_k0_key":   "  ",
			"extra_body_k0_value": "ignored",
			"extra_body_k1_key":   "keep",
			"extra_body_k1_value": "1",
		})
		m, err := parseExtraBodyFromForm(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m) != 1 || m["keep"] != int64(1) {
			t.Errorf("expected only keep=1, got %#v", m)
		}
	})

	t.Run("duplicate keys are rejected", func(t *testing.T) {
		r := makeRequest(map[string]string{
			"extra_body_count":    "2",
			"extra_body_k0_key":   "dup",
			"extra_body_k0_value": "1",
			"extra_body_k1_key":   "dup",
			"extra_body_k1_value": "2",
		})
		if _, err := parseExtraBodyFromForm(r); err == nil {
			t.Error("expected error for duplicate keys")
		}
	})

	t.Run("all-empty yields nil", func(t *testing.T) {
		r := makeRequest(map[string]string{
			"extra_body_count":  "1",
			"extra_body_k0_key": "",
		})
		m, err := parseExtraBodyFromForm(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m != nil {
			t.Errorf("expected nil, got %#v", m)
		}
	})
}
