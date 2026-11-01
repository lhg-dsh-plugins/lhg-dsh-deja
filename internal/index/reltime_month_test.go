package index

import (
	"testing"
	"time"
)

// The README offers `deja "what did we do in may"` as a feature, and the
// parser knew only relative phrases — "may" was searched as an ordinary word.
func TestMonthNamesResolveToTheMostRecentOccurrence(t *testing.T) {
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)
	for query, want := range map[string]string{
		"what did we do in may": "2026-05",
		"что делали в мае":      "2026-05",
		"in june":               "2026-06",
		"back in january":       "2026-01",
		// Asking about a month later than the current one means last year's:
		// nobody is asking what they did next December.
		"in december": "2025-12",
	} {
		got := relativeTimeTerms(query, now)
		found := false
		for _, tok := range got {
			if tok == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("%q -> %v, want %s among them", query, got, want)
		}
	}
	// A query with no time in it must stay empty, or every search picks up
	// month hints it never asked for.
	if got := relativeTimeTerms("fix the parser", now); len(got) != 0 {
		t.Fatalf("time hints invented for a plain query: %v", got)
	}
	// "March" as a verb, and "may" as a modal, are the cost of this: both
	// produce a hint. The tier only ever OR-scores them, so a wrong hint
	// costs ranking weight and never filters anything out.
	if got := relativeTimeTerms("we may need to fix this", now); len(got) == 0 {
		t.Log("modal 'may' produced no hint; fine either way")
	}
}

// "last month" asked on a 31st must still read the previous calendar month.
// AddDate(0,-1,0) normalizes 2026-10-31 forward to 2026-10-01, so the hint
// used to resolve to the current month's token and boost this month's
// sessions instead — the nightly failed on 2026-10-31 exactly this way.
func TestLastMonthSurvivesMonthEnd(t *testing.T) {
	for _, day := range []int{28, 29, 30, 31} {
		now := time.Date(2026, time.October, day, 9, 48, 0, 0, time.UTC)
		got := relativeTimeTerms("what did we do last month", now)
		found := false
		for _, tok := range got {
			if tok == "2026-09" {
				found = true
			}
			if tok == "2026-10" {
				t.Fatalf("oct %d: %q resolved \"last month\" to the current month: %v", day, now.Format("2006-01-02"), got)
			}
		}
		if !found {
			t.Fatalf("oct %d: no 2026-09 hint among %v", day, got)
		}
	}
}
