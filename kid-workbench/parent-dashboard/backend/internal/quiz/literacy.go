package quiz

// charPinyin 是识字表汉字的读音（不带声调）。
//
// 存在的唯一目的是排除同音字干扰项：「听读音选字」如果把「他」和「她」
// 放进同一道题，孩子答错不是因为不认字，而是这道题本身无解。
// 不带声调是刻意的——「目 mù」和「木 mù」连声调都一样，去声调只会更严格。
var charPinyin = map[string]string{
	"一": "yi", "二": "er", "三": "san", "四": "si", "五": "wu",
	"六": "liu", "七": "qi", "八": "ba", "九": "jiu", "十": "shi",

	"人": "ren", "口": "kou", "手": "shou", "足": "zu", "目": "mu",
	"耳": "er", "日": "ri", "月": "yue", "水": "shui", "火": "huo",

	"山": "shan", "石": "shi", "田": "tian", "土": "tu", "木": "mu",
	"禾": "he", "米": "mi", "竹": "zhu", "花": "hua", "草": "cao",

	"大": "da", "小": "xiao", "多": "duo", "少": "shao", "上": "shang",
	"下": "xia", "左": "zuo", "右": "you", "前": "qian", "后": "hou",

	"天": "tian", "地": "di", "风": "feng", "云": "yun", "雨": "yu",
	"雪": "xue", "春": "chun", "夏": "xia", "秋": "qiu", "冬": "dong",

	"爸": "ba", "妈": "ma", "爷": "ye", "奶": "nai", "哥": "ge",
	"姐": "jie", "弟": "di", "妹": "mei", "家": "jia", "我": "wo",

	"你": "ni", "他": "ta", "她": "ta", "们": "men", "的": "de",
	"了": "le", "不": "bu", "在": "zai", "有": "you", "是": "shi",

	"来": "lai", "去": "qu", "看": "kan", "听": "ting", "说": "shuo",
	"写": "xie", "读": "du", "画": "hua", "唱": "chang", "跳": "tiao",

	"红": "hong", "黄": "huang", "蓝": "lan", "绿": "lv", "白": "bai",
	"黑": "hei", "猫": "mao", "狗": "gou", "鸟": "niao", "鱼": "yu",

	"车": "che", "船": "chuan", "门": "men", "窗": "chuang", "书": "shu",
	"笔": "bi", "纸": "zhi", "课": "ke", "学": "xue", "玩": "wan",

	"牛": "niu", "羊": "yang", "马": "ma", "鸡": "ji", "鸭": "ya",
	"虫": "chong", "蜂": "feng", "蝶": "die", "蛙": "wa", "龟": "gui",

	"刀": "dao", "尺": "chi", "伞": "san", "灯": "deng", "床": "chuang",
	"桌": "zhuo", "椅": "yi", "碗": "wan", "勺": "shao", "杯": "bei",

	"金": "jin", "银": "yin", "铜": "tong", "铁": "tie", "线": "xian",
	"绳": "sheng", "包": "bao", "盒": "he", "瓶": "ping", "罐": "guan",

	"头": "tou", "脸": "lian", "牙": "ya", "鼻": "bi", "心": "xin",
	"笑": "xiao", "哭": "ku", "爱": "ai", "好": "hao", "乖": "guai",

	"衣": "yi", "帽": "mao", "鞋": "xie", "袜": "wa", "巾": "jin",
	"裤": "ku", "裙": "qun", "被": "bei", "枕": "zhen", "袋": "dai",

	"饭": "fan", "菜": "cai", "肉": "rou", "蛋": "dan", "茶": "cha",
	"果": "guo", "苹": "ping", "桃": "tao", "瓜": "gua", "豆": "dou",

	"早": "zao", "晚": "wan", "今": "jin", "明": "ming", "年": "nian",
	"友": "you", "园": "yuan", "班": "ban", "师": "shi", "生": "sheng",

	"走": "zou", "跑": "pao", "飞": "fei", "坐": "zuo", "站": "zhan",
	"星": "xing", "光": "guang", "电": "dian", "气": "qi", "开": "kai",

	"眼": "yan", "嘴": "zui", "舌": "she", "发": "fa", "脖": "bo",
	"肩": "jian", "臂": "bi", "指": "zhi", "肚": "du", "背": "bei",

	"树": "shu", "叶": "ye", "林": "lin", "河": "he", "湖": "hu",
	"海": "hai", "沙": "sha", "路": "lu", "桥": "qiao", "岛": "dao",

	"刷": "shua", "梳": "shu", "镜": "jing", "皂": "zao", "盆": "pen",
	"桶": "tong", "扫": "sao", "箱": "xiang", "柜": "gui", "锁": "suo",

	"里": "li", "外": "wai", "中": "zhong", "旁": "pang", "边": "bian",
	"东": "dong", "西": "xi", "南": "nan", "北": "bei", "方": "fang",

	"冷": "leng", "热": "re", "暖": "nuan", "凉": "liang", "晴": "qing",
	"阴": "yin", "出": "chu", "回": "hui", "进": "jin", "到": "dao",

	"兔": "tu", "鼠": "shu", "虎": "hu", "龙": "long", "蛇": "she",
	"猪": "zhu", "猴": "hou", "熊": "xiong", "狮": "shi", "象": "xiang",

	"面": "mian", "汤": "tang", "糖": "tang", "饼": "bing", "饺": "jiao",
	"油": "you", "盐": "yan", "醋": "cu", "酱": "jiang", "粥": "zhou",

	"本": "ben", "页": "ye", "字": "zi", "词": "ci", "句": "ju",
	"文": "wen", "诗": "shi", "歌": "ge", "音": "yin", "图": "tu",

	"昨": "zuo", "每": "mei", "次": "ci", "刚": "gang", "才": "cai",
	"正": "zheng", "快": "kuai", "慢": "man", "先": "xian", "又": "you",

	"请": "qing", "谢": "xie", "对": "dui", "起": "qi", "再": "zai",
	"见": "jian", "问": "wen", "答": "da", "叫": "jiao", "名": "ming",

	"吃": "chi", "喝": "he", "睡": "shui", "醒": "xing", "洗": "xi",
	"穿": "chuan", "脱": "tuo", "拿": "na", "给": "gei", "放": "fang",

	"雷": "lei", "雾": "wu", "冰": "bing", "虹": "hong", "霞": "xia",
	"露": "lu", "烟": "yan", "灰": "hui", "尘": "chen", "泥": "ni",
}

func literacySpecs(kp Kp) []Spec {
	char := kp.Title
	py, ok := charPinyin[char]
	if !ok {
		return nil
	}

	pool := make([]string, 0, len(kp.Siblings))
	for _, s := range kp.Siblings {
		if s == char {
			continue
		}
		if p, ok := charPinyin[s]; !ok || p == py {
			continue
		}
		pool = append(pool, s)
	}
	if len(pool) < optionCount-1 {
		return nil
	}

	// 两种题型各一题；义图资源未接入前，字图/义图题干与选项先用大字卡占位，保证布局与掌握维度可跑通。
	specs := make([]Spec, 0, 2)

	opts, idx := labelOptions(char, [][]string{pool}, rngFor(kp.ID, 2))
	specs = append(specs, Spec{
		Code: "glyph_sense", Stem: "看字图，选出义图（暂用字卡）",
		Options: opts, AnswerIndex: idx,
		Visual: Visual{Kind: "char", Text: char},
		Speech: Speech{Text: char, Lang: LangZH},
	})

	opts, idx = labelOptions(char, [][]string{pool}, rngFor(kp.ID, 3))
	specs = append(specs, Spec{
		Code: "sense_char", Stem: "看义图，选出字（暂用字卡）",
		Options: opts, AnswerIndex: idx,
		Visual: Visual{Kind: "char", Text: char},
		Speech: Speech{Text: char, Lang: LangZH},
	})

	return specs
}
