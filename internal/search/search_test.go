package search_test

import (
	"testing"

	"github.com/ChengaDev/persist/internal/search"
)

func TestFilter(t *testing.T) {
	keys := []string{"github-token", "github-key", "db-password", "aws-secret"}

	got := search.Filter("github", keys)
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %v", got)
	}

	// Case-insensitive
	got = search.Filter("GITHUB", keys)
	if len(got) != 2 {
		t.Fatalf("expected case-insensitive match, got %v", got)
	}

	got = search.Filter("notfound", keys)
	if len(got) != 0 {
		t.Fatalf("expected no results, got %v", got)
	}
}

func TestFilterEmpty(t *testing.T) {
	got := search.Filter("anything", nil)
	if len(got) != 0 {
		t.Fatal("expected empty result for nil candidates")
	}
}

func TestSuggestSubstringMatch(t *testing.T) {
	keys := []string{"github-token", "github-key", "db-password"}

	got := search.Suggest("github", keys, 3)
	if len(got) == 0 {
		t.Fatal("expected substring matches")
	}
	for _, k := range got {
		if k != "github-token" && k != "github-key" {
			t.Fatalf("unexpected suggestion: %q", k)
		}
	}
}

func TestSuggestTypo(t *testing.T) {
	keys := []string{"github-token", "db-password", "aws-secret"}

	// One character off
	got := search.Suggest("github-tokne", keys, 3)
	if len(got) == 0 || got[0] != "github-token" {
		t.Fatalf("expected github-token as top suggestion, got %v", got)
	}
}

func TestSuggestNoCloseMatch(t *testing.T) {
	keys := []string{"alpha", "beta", "gamma"}

	got := search.Suggest("zzzzzzzzz", keys, 3)
	if len(got) != 0 {
		t.Fatalf("expected no suggestions for completely unrelated query, got %v", got)
	}
}

func TestSuggestRespectsMaxResults(t *testing.T) {
	keys := []string{"github-token", "github-key", "github-secret", "github-id"}

	got := search.Suggest("github", keys, 2)
	if len(got) > 2 {
		t.Fatalf("expected at most 2 results, got %d", len(got))
	}
}

func TestSuggestSubstringRanksAboveEditDistance(t *testing.T) {
	keys := []string{"totally-unrelated-x", "github-token"}

	// "github" is a substring of "github-token" but not "totally-unrelated-x"
	got := search.Suggest("github", keys, 2)
	if len(got) == 0 || got[0] != "github-token" {
		t.Fatalf("substring match should rank first, got %v", got)
	}
}
