package gorm_test

import (
	"testing"
	"time"

	gorma "github.com/bornholm/xolo/internal/adapter/gorm"
	"github.com/bornholm/xolo/internal/core/model"
)

func TestJSONColumn_NilValue(t *testing.T) {
	col := gorma.JSONColumn[model.RetryConfig]{}
	v, err := col.Value()
	if err != nil {
		t.Fatal(err)
	}
	if v != nil {
		t.Errorf("expected nil driver.Value for zero JSONColumn, got %v", v)
	}
}

func TestJSONColumn_RoundTrip(t *testing.T) {
	want := model.RetryConfig{
		Enabled:     true,
		MaxAttempts: 3,
		Delay:       2 * time.Second,
	}
	col := gorma.JSONColumn[model.RetryConfig]{Val: &want}

	v, err := col.Value()
	if err != nil {
		t.Fatal(err)
	}

	var col2 gorma.JSONColumn[model.RetryConfig]
	if err := col2.Scan(v); err != nil {
		t.Fatal(err)
	}
	if col2.Val == nil {
		t.Fatal("Scan: expected non-nil Val")
	}
	if *col2.Val != want {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", *col2.Val, want)
	}
}

func TestJSONColumn_ScanNil(t *testing.T) {
	var col gorma.JSONColumn[model.RetryConfig]
	if err := col.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if col.Val != nil {
		t.Error("Scan(nil): expected nil Val")
	}
}

func TestJSONColumn_ScanBytes(t *testing.T) {
	want := model.RateLimitConfig{Enabled: true, Interval: time.Minute, MaxBurst: 5}
	col := gorma.JSONColumn[model.RateLimitConfig]{Val: &want}

	v, err := col.Value()
	if err != nil {
		t.Fatal(err)
	}
	var col2 gorma.JSONColumn[model.RateLimitConfig]
	if err := col2.Scan([]byte(v.(string))); err != nil {
		t.Fatal(err)
	}
	if col2.Val == nil || *col2.Val != want {
		t.Errorf("Scan([]byte) mismatch: got %+v, want %+v", col2.Val, want)
	}
}
