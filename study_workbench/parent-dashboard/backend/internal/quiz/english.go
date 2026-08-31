package quiz

// englishEmoji 只收录含义明确、不会引起歧义的 emoji。
// 没收录的单词（kind、welcome、yesterday 这类抽象词）不会出"看图选词"，
// 宁可少一种题型，也不要让孩子对着一个猜不出含义的图标发呆。
var englishEmoji = map[string]string{
	"cat": "🐱", "dog": "🐶", "bird": "🐦", "fish": "🐟", "rabbit": "🐰",
	"tiger": "🐯", "lion": "🦁", "bear": "🐻", "monkey": "🐵", "panda": "🐼",

	"red": "🟥", "blue": "🟦", "green": "🟩", "yellow": "🟨",
	"black": "⬛", "white": "⬜", "orange": "🟧", "purple": "🟪", "brown": "🟫",

	"one": "1️⃣", "two": "2️⃣", "three": "3️⃣", "four": "4️⃣", "five": "5️⃣",
	"six": "6️⃣", "seven": "7️⃣", "eight": "8️⃣", "nine": "9️⃣", "ten": "🔟",

	"apple": "🍎", "banana": "🍌", "bread": "🍞", "milk": "🥛", "egg": "🥚",
	"rice": "🍚", "cake": "🍰", "juice": "🧃", "soup": "🍲", "candy": "🍬",

	"mom": "👩", "dad": "👨", "baby": "👶", "grandpa": "👴", "grandma": "👵",
	"brother": "👦", "sister": "👧",

	"eye": "👁️", "ear": "👂", "nose": "👃", "mouth": "👄", "hand": "✋",
	"foot": "🦶", "arm": "💪", "leg": "🦵", "hair": "💇",

	"grape": "🍇", "peach": "🍑", "pear": "🍐", "lemon": "🍋", "melon": "🍈",
	"cherry": "🍒", "mango": "🥭", "kiwi": "🥝", "berry": "🫐",

	"sunny": "☀️", "rainy": "🌧️", "cloudy": "☁️", "windy": "💨",
	"snowy": "❄️", "storm": "⛈️", "hot": "🥵", "cold": "🥶",

	"ball": "⚽", "doll": "🪆", "block": "🧱", "kite": "🪁",
	"puzzle": "🧩", "robot": "🤖", "balloon": "🎈", "slide": "🛝",

	"book": "📖", "pen": "🖊️", "pencil": "✏️", "bag": "🎒", "chair": "🪑",
	"teacher": "👩‍🏫", "student": "🧑‍🎓", "school": "🏫",

	"shirt": "👕", "pants": "👖", "dress": "👗", "hat": "🎩", "shoe": "👟",
	"sock": "🧦", "coat": "🧥", "scarf": "🧣", "glove": "🧤",

	"run": "🏃", "jump": "🤸", "walk": "🚶", "stand": "🧍", "eat": "🍽️",
	"drink": "🥤", "sleep": "😴", "read": "📚", "write": "✍️",

	"home": "🏠", "park": "🏞️", "zoo": "🦓", "shop": "🏪", "farm": "🚜",
	"beach": "🏖️", "library": "📕", "museum": "🏛️", "hospital": "🏥", "cinema": "🎬",

	"bus": "🚌", "bike": "🚲", "plane": "✈️", "boat": "⛵", "train": "🚂",
	"taxi": "🚕", "truck": "🚚", "subway": "🚇", "helicopter": "🚁", "ship": "🚢",

	"circle": "⭕", "square": "🔷", "triangle": "🔺", "star": "⭐",
	"heart": "❤️", "diamond": "🔶", "arrow": "➡️", "cross": "✖️",

	// time 话题刻意不给 emoji：🌅 日出、🌇 日落、🌆 黄昏连大人都分不清，
	// 让孩子在这四个图里选只会变成猜。这个话题只出听音选词。

	"happy": "😀", "sad": "😢", "angry": "😠", "tired": "😪",
	"scared": "😨", "funny": "😂",

	"tree": "🌳", "flower": "🌸", "grass": "🌿", "river": "🌊", "mountain": "⛰️",
	"sun": "🌤️", "moon": "🌙", "cloud": "☁️", "rain": "🌧️",

	"doctor": "👨‍⚕️", "nurse": "👩‍⚕️", "chef": "👨‍🍳", "pilot": "👨‍✈️",
	"farmer": "👨‍🌾", "singer": "🎤", "dancer": "💃", "police": "👮",
	"firefighter": "🧑‍🚒", "driver": "🚗",
}

func englishSpecs(kp Kp) []Spec {
	word := kp.Title

	// 干扰项只从同话题单词里取。跨话题抽（cat 的干扰项给 hospital）
	// 孩子能靠"不是一类东西"排除，测不出是否真的听懂了单词。
	pool := make([]string, 0, len(kp.Siblings))
	for _, s := range kp.Siblings {
		if s != word {
			pool = append(pool, s)
		}
	}
	if len(pool) < optionCount-1 {
		return nil
	}

	opts, idx := labelOptions(word, [][]string{pool}, rngFor(kp.ID, 1))
	specs := []Spec{{
		Code: "listen", Stem: "听一听，点出这个单词",
		Options: opts, AnswerIndex: idx,
		Speech: Speech{Text: word, Lang: LangEN},
	}}

	if spec, ok := pictureSpec(kp, word, pool); ok {
		specs = append(specs, spec)
	}
	return specs
}

// pictureSpec 出"听单词选图"，要求正确项和至少 3 个同话题干扰项都有 emoji，
// 且 emoji 互不重复——选项里出现两个一样的图，题目就废了。
func pictureSpec(kp Kp, word string, pool []string) (Spec, bool) {
	own, ok := englishEmoji[word]
	if !ok {
		return Spec{}, false
	}

	var candidates []string
	seenEmoji := map[string]bool{own: true}
	for _, s := range pool {
		e, ok := englishEmoji[s]
		if !ok || seenEmoji[e] {
			continue
		}
		seenEmoji[e] = true
		candidates = append(candidates, s)
	}
	if len(candidates) < optionCount-1 {
		return Spec{}, false
	}

	rng := rngFor(kp.ID, 2)
	picked := pickTiered([][]string{candidates}, optionCount-1, func(string) bool { return false }, rng)

	ds := make([]Option, 0, len(picked))
	for _, p := range picked {
		ds = append(ds, Option{Emoji: englishEmoji[p]})
	}
	opts, idx := buildOptions(Option{Emoji: own}, ds, rng)

	return Spec{
		Code: "picture", Stem: "听一听，点出对应的图",
		Options: opts, AnswerIndex: idx,
		Speech: Speech{Text: word, Lang: LangEN},
	}, true
}
