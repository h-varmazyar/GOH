package tse

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var persianReplacer = strings.NewReplacer(
	"ي", "ی", "ى", "ی", "ئ", "ی",
	"ك", "ک",
	"ة", "ه", "ۀ", "ه",
	"أ", "ا", "إ", "ا", "ٱ", "ا",
	"ؤ", "و",
	"ـ", "",
	"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
)

var displayReplacer = strings.NewReplacer("ي", "ی", "ى", "ی", "ك", "ک")

// cleanText converts Arabic presentation forms and Arabic letters to their
// Persian equivalents and collapses whitespace. The result is meant for output.
func cleanText(value string) string {
	value = displayReplacer.Replace(norm.NFKC.String(value))
	return strings.Join(strings.Fields(value), " ")
}

// matchKey reduces a symbol or name to a form that is equal for spelling
// variants used by TSE, IFB and TSETMC (Arabic letters, ZWNJ, spaces, punctuation).
func matchKey(value string) string {
	value = persianReplacer.Replace(norm.NFKC.String(value))
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// nameSimilarity returns the Dice coefficient of the character bigrams of two
// names, in the range [0, 1]. A name contained in the other one scores 1.
func nameSimilarity(a, b string) float64 {
	a, b = matchKey(a), matchKey(b)
	if a == "" || b == "" {
		return 0
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 1
	}
	bigrams := func(value string) map[string]int {
		runes := []rune(value)
		result := make(map[string]int, len(runes))
		for i := 0; i+1 < len(runes); i++ {
			result[string(runes[i:i+2])]++
		}
		return result
	}
	left, right := bigrams(a), bigrams(b)
	total, common := 0, 0
	for gram, count := range left {
		total += count
		common += min(count, right[gram])
	}
	for _, count := range right {
		total += count
	}
	if total == 0 {
		return 0
	}
	return 2 * float64(common) / float64(total)
}

// SymbolKey returns the form of a symbol under which spelling variants of the
// same symbol are equal.
func SymbolKey(symbol string) string { return matchKey(symbol) }

// NameSimilarity returns the similarity of two instrument names in the range [0, 1].
func NameSimilarity(a, b string) float64 { return nameSimilarity(a, b) }

// companyNameNoise are words that TSE sources add to or drop from company names.
var companyNameNoise = []string{"حذف", "شرکت"}

// CompanyNameKey returns the form of a company name under which the names that
// different sources give the same company are equal. Unlike NameSimilarity it
// only matches names that are equal apart from spelling.
func CompanyNameKey(name string) string {
	key := matchKey(name)
	for _, prefix := range companyNameNoise {
		key = strings.TrimPrefix(key, matchKey(prefix))
	}
	return strings.ReplaceAll(key, matchKey("سهامی عام"), "")
}
