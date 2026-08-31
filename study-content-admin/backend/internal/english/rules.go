package english

// NeedsSenseImage reports whether the English word should get a sense illustration.
func NeedsSenseImage(word string) bool {
	if _, no := noSenseImageWords[word]; no {
		return false
	}
	return true
}

// Abstract / function words without a stable drawable object.
var noSenseImageWords = map[string]struct{}{
	"hello": {}, "hi": {}, "bye": {}, "thanks": {}, "please": {}, "sorry": {},
	"yes": {}, "no": {}, "ok": {},
	"today": {}, "yesterday": {}, "tomorrow": {}, "week": {}, "year": {},
	"kind": {},
}
