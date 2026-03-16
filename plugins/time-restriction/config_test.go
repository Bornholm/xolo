package main

import (
	"testing"
	"time"
)

// monday09h30 est un lundi à 09h30 UTC (2026-03-16 est un lundi).
var monday09h30 = time.Date(2026, 3, 16, 9, 30, 0, 0, time.UTC)

func cfg(timezone string, slots ...Slot) Config {
	return Config{Timezone: timezone, Slots: slots}
}

func slot(start, end string, days ...string) Slot {
	return Slot{Days: days, Start: start, End: end}
}

func TestIsAllowed_MatchingSlot(t *testing.T) {
	c := cfg("UTC", slot("09:00", "18:00", "monday"))
	if !isAllowed(monday09h30, c) {
		t.Fatal("expected allowed")
	}
}

func TestIsAllowed_BeforeSlot(t *testing.T) {
	c := cfg("UTC", slot("10:00", "18:00", "monday"))
	if isAllowed(monday09h30, c) {
		t.Fatal("expected denied: before start")
	}
}

func TestIsAllowed_AfterSlot(t *testing.T) {
	c := cfg("UTC", slot("09:00", "09:15", "monday"))
	if isAllowed(monday09h30, c) {
		t.Fatal("expected denied: after end")
	}
}

func TestIsAllowed_WrongDay(t *testing.T) {
	c := cfg("UTC", slot("09:00", "18:00", "tuesday", "wednesday"))
	if isAllowed(monday09h30, c) {
		t.Fatal("expected denied: wrong day")
	}
}

func TestIsAllowed_EmptySlots(t *testing.T) {
	c := cfg("UTC")
	if isAllowed(monday09h30, c) {
		t.Fatal("expected denied: empty slots")
	}
}

func TestIsAllowed_EmptyConfig(t *testing.T) {
	if isAllowed(monday09h30, Config{}) {
		t.Fatal("expected denied: zero-value config")
	}
}

func TestIsAllowed_InvalidTimezone(t *testing.T) {
	c := cfg("Invalid/Zone", slot("09:00", "18:00", "monday"))
	if isAllowed(monday09h30, c) {
		t.Fatal("expected denied: invalid timezone")
	}
}

func TestIsAllowed_EndExclusive(t *testing.T) {
	// 09:30 == end → doit être refusé (end est exclusif)
	c := cfg("UTC", slot("09:00", "09:30", "monday"))
	if isAllowed(monday09h30, c) {
		t.Fatal("expected denied: now == end (exclusive)")
	}
}

func TestIsAllowed_MultipleSlots(t *testing.T) {
	c := cfg("UTC",
		slot("07:00", "08:00", "monday"), // ne matche pas (trop tôt)
		slot("09:00", "18:00", "monday"), // matche
	)
	if !isAllowed(monday09h30, c) {
		t.Fatal("expected allowed: second slot matches")
	}
}

func TestParseConfig_InvalidJSON(t *testing.T) {
	_, err := parseConfig("{invalid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// empty string means "no configuration saved yet" and must NOT be treated as
// a JSON error — parseConfig("") should return (Config{}, nil).
func TestParseConfig_EmptyString(t *testing.T) {
	got, err := parseConfig("")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(got.Slots) != 0 || got.Timezone != "" {
		t.Fatalf("expected zero Config, got: %+v", got)
	}
}
