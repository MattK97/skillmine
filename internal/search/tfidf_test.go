package search

import (
	"os"
	"testing"

	"github.com/MattK97/skillmine/internal/transcript"
)

// targets are the seven prompts in testdata/sample.jsonl that all ask for the
// same thing in different words. They sit among 32 unrelated prompts, several
// of which deliberately share database vocabulary — without those distractors
// the test would measure nothing.
//
// Known limit: "zero out all the tables so we can start from scratch" shares
// almost no vocabulary with the others. A lexical method cannot reach it from
// a query phrased like the rest, which is why the expectations below stop
// short of 7.
var targets = map[string]bool{
	"t1": true, "t2": true, "t3": true, "t4": true,
	"t5": true, "t6": true, "t7": true,
}

// minScore is the default cutoff, matching the -min-score flag.
// It must stay above zero: at zero, documents with no shared terms enter the
// result set and any recall assertion passes for the wrong reason.
const minScore = 0.10

// TestGolden is the quality bar for the whole tool. It asserts behaviour, not
// implementation, so the similarity method can be replaced wholesale and this
// test still says whether the replacement is better.
func TestGolden(t *testing.T) {
	idx := loadFixture(t)

	tests := []struct {
		name     string
		query    string
		wantMin  int
		cleanTop int // this many leading results must all be targets
	}{
		{
			name:     "short query",
			query:    "reset the database",
			wantMin:  4,
			cleanTop: 3,
		},
		{
			name:     "query enriched from context",
			query:    "reset the test database keep the user account",
			wantMin:  5,
			cleanTop: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits := idx.Search(tt.query, 12, minScore)

			found := 0
			for _, h := range hits {
				if targets[h.Record.UUID] {
					found++
				}
			}
			if found < tt.wantMin {
				t.Errorf("found %d of %d targets, want at least %d", found, len(targets), tt.wantMin)
				dump(t, hits)
			}

			// Recall alone is not enough: the top of the ranking is what a
			// human or a model actually reads.
			if len(hits) < tt.cleanTop {
				t.Fatalf("only %d hits, cannot check the top %d", len(hits), tt.cleanTop)
			}
			for i := 0; i < tt.cleanTop; i++ {
				if !targets[hits[i].Record.UUID] {
					t.Errorf("rank %d is not a target: %q", i+1, hits[i].Record.Text)
					dump(t, hits)
					break
				}
			}
		})
	}
}

func TestNoMatches(t *testing.T) {
	idx := loadFixture(t)
	if hits := idx.Search("compiling a freebsd kernel on sparc", 10, minScore); len(hits) > 0 {
		t.Errorf("want no hits, got %d", len(hits))
		dump(t, hits)
	}
}

func TestEmptyQuery(t *testing.T) {
	idx := loadFixture(t)
	if hits := idx.Search("   ", 10, 0); hits != nil {
		t.Errorf("empty query returned %d hits", len(hits))
	}
}

func TestResultsAreOrdered(t *testing.T) {
	idx := loadFixture(t)
	hits := idx.Search("reset the database", 10, 0)
	for i := 1; i < len(hits); i++ {
		if hits[i-1].Score < hits[i].Score {
			t.Fatalf("hit %d scores %.3f, below hit %d at %.3f", i-1, hits[i-1].Score, i, hits[i].Score)
		}
	}
}

func TestTopZeroReturnsEverything(t *testing.T) {
	idx := loadFixture(t)
	limited := idx.Search("reset the database", 2, minScore)
	all := idx.Search("reset the database", 0, minScore)
	if len(limited) != 2 {
		t.Fatalf("top=2 returned %d hits", len(limited))
	}
	if len(all) <= len(limited) {
		t.Errorf("top=0 returned %d hits, expected more than %d", len(all), len(limited))
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "case and diacritics folded",
			in:   "Wyczyść Bazę",
			want: []string{"wyczys", "baze"},
		},
		{
			// NFD leaves these eight alone; the replacer table catches them.
			name: "letters unicode normalisation does not decompose",
			in:   "łóż straße æon øre",
			want: []string{"loz", "strass", "aeon", "ore"},
		},
		{
			// Beyond the languages a hand-written table would have covered.
			name: "turkish, vietnamese, baltic",
			in:   "değil tiếng šešios",
			want: []string{"degil", "tieng", "sesios"},
		},
		{
			name: "inflections collapse to a shared stem",
			in:   "conversation conversations",
			want: []string{"conver", "conver"},
		},
		{
			name: "paths split into parts, short tokens dropped",
			in:   "lib/src/data/dto.dart:55",
			want: []string{"lib", "src", "data", "dto", "dart"},
		},
		{
			name: "punctuation only",
			in:   "-- :: !!",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("tokenize(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func loadFixture(t *testing.T) *Index {
	t.Helper()

	f, err := os.Open("../../testdata/sample.jsonl")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()

	recs, err := transcript.ScanFile(f, "demo-app")
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	prompts := transcript.Prompts(recs)
	if len(prompts) < 30 {
		t.Fatalf("fixture has only %d prompts — too few distractors to measure anything", len(prompts))
	}
	return New(prompts)
}

// dump prints the ranking so a failure shows what the tool considered similar.
func dump(t *testing.T, hits []Hit) {
	t.Helper()
	for i, h := range hits {
		mark := "        "
		if targets[h.Record.UUID] {
			mark = "TARGET  "
		}
		t.Logf("  %2d. %.3f %s %s", i+1, h.Score, mark, transcript.Shorten(h.Record.Text, 70))
	}
}
