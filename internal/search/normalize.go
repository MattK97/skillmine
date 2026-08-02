package search

import "strings"

// stemLen is how many characters a token is truncated to.
//
// A crude prefix stemmer, but it collapses most inflected forms across the
// languages people actually type prompts in ("cleanup"/"cleaning" -> "cleani",
// Polish "wyczyscic"/"wyczyscil" -> "wyczys") without pulling in a real
// morphological stemmer as a dependency.
const stemLen = 6

// foldDiacritics maps accented Latin characters to their base form, so that
// "wyczyść bazę" and "wyczysc baze" tokenize identically. People are
// inconsistent about typing accents, especially when typing fast.
var foldDiacritics = strings.NewReplacer(
	// Polish
	"ą", "a", "ć", "c", "ę", "e", "ł", "l", "ń", "n",
	"ó", "o", "ś", "s", "ź", "z", "ż", "z",
	// German, Scandinavian
	"ä", "a", "ö", "o", "ü", "u", "ß", "ss", "å", "a", "æ", "ae", "ø", "o",
	// Romance
	"á", "a", "à", "a", "â", "a", "ã", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ò", "o", "ô", "o", "õ", "o",
	"ú", "u", "ù", "u", "û", "u",
	"ñ", "n", "ç", "c",
	// Czech, Slovak, Croatian
	"č", "c", "ď", "d", "ě", "e", "ň", "n", "ř", "r",
	"š", "s", "ť", "t", "ů", "u", "ž", "z",
)

// stopWords carries no topical information. TF-IDF already suppresses common
// words, so this is mostly a memory optimisation plus insurance for small
// corpora where IDF has little to work with.
//
// Add your own language here; unknown words are simply kept.
var stopWords = map[string]bool{}

func init() {
	const english = `
		the and are but can could did does for from had has have her his its
		not our that their them then there these they this those was were what
		when which will with would you your just also need want please
		make sure like get got put set now then here about into over
	`
	const polish = `
		aby ale albo bez być była było były będzie dla gdy gdzie ich jak
		jakie jego jej jest jestem już która które który lub mam może nad nie
		niż pod przez przy się tak także tam tego tej ten teraz też tutaj
		tylko tym więc wszystko żeby jeszcze trochę bardzo chcę możesz
	`
	// Conversational filler that shows up constantly in chat-style prompts.
	const filler = `
		okay yeah yep nope hmm ehh dobra spoko kurcze wiesz sensie generalnie znowu
	`

	for _, w := range strings.Fields(english + polish + filler) {
		stopWords[fold(w)] = true
	}
}

func fold(s string) string { return foldDiacritics.Replace(strings.ToLower(s)) }

// tokenize turns text into comparable stems.
func tokenize(s string) []string {
	folded := fold(s)

	// Split on anything that is not an ASCII letter or digit. Paths, markup and
	// punctuation fall apart into their component words as a side effect.
	words := strings.FieldsFunc(folded, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})

	out := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) < 3 || stopWords[w] {
			continue
		}
		if len(w) > stemLen {
			w = w[:stemLen]
		}
		out = append(out, w)
	}
	return out
}
