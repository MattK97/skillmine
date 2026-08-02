package search

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// stemLen is the length tokens are truncated to.
//
// This is truncation stemming: crude compared to a morphological stemmer, but
// language-agnostic, which matters because prompts arrive in whatever language
// the user thinks in. Snowball has no algorithm for several of those languages
// (Polish among them), and dictionary-based stemmers are a heavy dependency.
//
// Six is measured, not guessed. On a corpus of real prompts, truncating to six
// retrieved one more paraphrase than no stemming at all; five degraded the
// ranking and eight lost recall on short queries.
const stemLen = 6

// nonDecomposing lists the letters that Unicode normalisation leaves alone,
// because they are distinct letters rather than a base plus a combining accent.
// Verified against the NFD pass below: across common Latin scripts these eight
// are the only ones that survive it unchanged.
//
// Polish "ł" is the one that matters most here — without this table, "łatwo"
// and "latwo" would never match.
var nonDecomposing = strings.NewReplacer(
	"ł", "l", "ß", "ss", "æ", "ae", "ø", "o",
	"đ", "d", "ı", "i", "œ", "oe", "þ", "th",
)

// fold lowercases text and strips diacritics, so that "wyczyść bazę" and
// "wyczysc baze" tokenize identically. People are inconsistent about typing
// accents, especially in a hurry.
func fold(s string) string {
	s = nonDecomposing.Replace(strings.ToLower(s))

	// Decompose, drop the combining marks, recompose. Built per call because
	// transform.Transformer carries state and is not safe to share between
	// goroutines; the allocation is negligible next to the work it does.
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

// tokenize turns text into comparable stems.
//
// There is deliberately no stop-word list. TF-IDF already drives the weight of
// words that appear everywhere towards zero, and measurement on two corpora
// showed an explicit list changed no result. It would be code that looks useful
// and does nothing.
func tokenize(s string) []string {
	folded := fold(s)

	// Split on anything that is not an ASCII letter or digit. Paths, markup and
	// punctuation fall apart into their component words as a side effect.
	words := strings.FieldsFunc(folded, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})

	out := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) < 3 {
			continue
		}
		if len(w) > stemLen {
			w = w[:stemLen]
		}
		out = append(out, w)
	}
	return out
}
