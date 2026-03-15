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
