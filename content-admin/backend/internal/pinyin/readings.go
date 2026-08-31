package pinyin

// Reading provides Solo (standalone pronunciation) and Word (example character) texts for TTS.
// Copied from study_workbench quiz/pinyin.go pinyinTable — keep in sync for MVP.
type Reading struct {
	Solo string
	Word string
}

// Readings maps pinyin letter (kp title) → Solo/Word. eng has empty Solo.
var Readings = map[string]Reading{
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
