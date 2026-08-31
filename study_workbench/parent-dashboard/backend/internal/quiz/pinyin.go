package quiz

// pinyinReading 提供两种朗读素材。
//
// 不能直接把字母交给 TTS：zh-CN 语音会把 "b" 念成英文字母名 "bee"，
// 把 "ang" 念成拼读乱码。所以用汉字承载读音——
//
//	Solo 是该字母单独的标准读音（声母 b 读"波"，韵母 ang 读"昂"）
//	Word 是包含该音的例字（b → 爸，ang → 羊），用来出"这个字里有哪个音"
type pinyinReading struct {
	Solo string
	Word string
}

// eng 没有可靠的单读汉字（"鞥"过于生僻，TTS 常读错），Solo 留空，
// 只出例字变体。宁可少一种题型，也不要让孩子听到错的读音。
var pinyinTable = map[string]pinyinReading{
	// 声母
	"b": {"波", "爸"}, "p": {"坡", "怕"}, "m": {"摸", "妈"}, "f": {"佛", "飞"},
	"d": {"得", "大"}, "t": {"特", "题"}, "n": {"呢", "你"}, "l": {"勒", "来"},
	"g": {"哥", "高"}, "k": {"科", "看"}, "h": {"喝", "好"},
	"j": {"基", "家"}, "q": {"欺", "去"}, "x": {"希", "小"},
	"zh": {"知", "只"}, "ch": {"吃", "车"}, "sh": {"诗", "书"}, "r": {"日", "肉"},
	"z": {"资", "走"}, "c": {"次", "菜"}, "s": {"思", "四"},
	"y": {"衣", "羊"}, "w": {"乌", "我"},

	// 韵母
	"a": {"啊", "妈"}, "o": {"喔", "我"}, "e": {"鹅", "鹅"},
	"i": {"衣", "米"}, "u": {"乌", "鼓"}, "ü": {"鱼", "鱼"},
	"ai": {"哀", "白"}, "ei": {"诶", "飞"}, "ui": {"威", "水"},
	"ao": {"熬", "猫"}, "ou": {"欧", "头"}, "iu": {"优", "牛"},
	"ie": {"耶", "叶"}, "üe": {"约", "月"}, "er": {"儿", "儿"},
	"an": {"安", "山"}, "en": {"恩", "门"}, "in": {"因", "心"},
	"un": {"温", "春"}, "ün": {"晕", "云"},
	"ang": {"昂", "羊"}, "eng": {"", "灯"},
}

// 干扰项取自易混组：形近（b/d/p/q）、音近（zh/ch/sh/r）、鼻音前后（an/ang）。
// 随机抽字母会让孩子靠"看起来不像"排除，测不出真实掌握度。
var pinyinConfusion = [][]string{
	{"b", "d", "p", "q"},
	{"m", "n", "f", "h"},
	{"g", "k", "h", "j"},
	{"j", "q", "x", "y"},
	{"zh", "ch", "sh", "r"},
	{"z", "c", "s", "zh"},
	{"t", "d", "l", "n"},
	{"y", "w", "m", "n"},
	{"a", "o", "e", "i"},
	{"i", "u", "ü", "e"},
	{"ai", "ei", "ui", "ao"},
	{"ao", "ou", "iu", "ai"},
	{"ie", "üe", "er", "iu"},
	{"an", "en", "in", "un"},
	{"un", "ün", "in", "en"},
	{"ang", "eng", "an", "en"},
}

func pinyinSpecs(kp Kp) []Spec {
	letter := kp.Title
	reading, ok := pinyinTable[letter]
	if !ok {
		return nil
	}

	tiers := confusionTiers(letter, kp.Siblings)
	if tierSize(tiers) < optionCount-1 {
		return nil
	}

	var specs []Spec
	if reading.Word != "" {
		opts, idx := labelOptions(letter, tiers, rngFor(kp.ID, 1))
		specs = append(specs, Spec{
			Code: "inword", Stem: "听一听，这个字里有哪个音？",
			Options: opts, AnswerIndex: idx,
			Visual: Visual{Kind: "char", Text: reading.Word},
			Speech: Speech{Text: reading.Word, Lang: LangZH},
		})
	}
	if reading.Solo != "" {
		opts, idx := labelOptions(letter, tiers, rngFor(kp.ID, 2))
		specs = append(specs, Spec{
			Code: "listen", Stem: "听一听，这是哪个字母？",
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: reading.Solo, Lang: LangZH},
		})
	}
	return specs
}

// confusionTiers 第一层是易混字母，第二层是同模块其他字母（易混组不足 3 个时兜底）。
func confusionTiers(letter string, siblings []string) [][]string {
	var confusing, rest []string
	seen := map[string]bool{letter: true}

	for _, group := range pinyinConfusion {
		if !contains(group, letter) {
			continue
		}
		for _, v := range group {
			if !seen[v] {
				seen[v] = true
				confusing = append(confusing, v)
			}
		}
	}
	for _, s := range siblings {
		if !seen[s] {
			seen[s] = true
			rest = append(rest, s)
		}
	}
	return [][]string{confusing, rest}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
