package seed

import "fmt"

type logicItem struct {
	Code  string
	Title string
	Kind  string
	// Pattern / Diff：题干上方的 emoji 序列
	Seq []string
	// 正确答案：pattern 是下一个，classify/diff 是被点出的那项，order 是排好的结果文字
	A     string
	Wrong []string
	// 读题文本；空则用默认
	Speech string
	Prompt string // 题干文字
	Diff   int
}

func logicKPs(prefix string, items []logicItem) []kpSpec {
	out := make([]kpSpec, 0, len(items))
	for i, it := range items {
		diff := it.Diff
		if diff == 0 {
			diff = 1
		}
		prompt := it.Prompt
		if prompt == "" {
			switch it.Kind {
			case "pattern":
				prompt = "下一个是哪个？"
			case "classify":
				prompt = "哪个和其他不一样？"
			case "order":
				prompt = "正确的顺序是？"
			case "diff":
				prompt = "找出不一样的那个"
			default:
				prompt = "选一选"
			}
		}
		speech := it.Speech
		if speech == "" {
			speech = prompt
		}
		out = append(out, kpSpec{
			Code: fmt.Sprintf("%s%03d", prefix, i+1), Title: it.Title, Difficulty: diff,
			Payload: mustPayload(map[string]any{
				"kind": it.Kind, "seq": it.Seq, "a": it.A, "wrong": it.Wrong,
				"prompt": prompt, "speech": speech,
			}),
		})
	}
	return out
}

func logicModules() []moduleSpec {
	return []moduleSpec{
		{Code: "pattern", Name: "找规律", Kps: logicKPs("pt", patternItems())},
		{Code: "classify", Name: "分类", Kps: logicKPs("cl", classifyItems())},
		{Code: "order", Name: "排序", Kps: logicKPs("od", orderItems())},
		{Code: "shape_reason", Name: "图形推理", Kps: logicKPs("sr", shapeReasonItems())},
		{Code: "diff", Name: "找不同", Kps: logicKPs("df", diffItems())},
	}
}

func patternItems() []logicItem {
	return []logicItem{
		{"pt01", "红蓝交替", "pattern", []string{"🔴", "🔵", "🔴", "🔵"}, "🔴", []string{"🟢", "🟡", "⬛"}, "", "", 1},
		{"pt02", "大小交替", "pattern", []string{"⬛", "⬜", "⬛", "⬜"}, "⬛", []string{"⬜", "🔺", "🔵"}, "", "", 1},
		{"pt03", "水果两个一换", "pattern", []string{"🍎", "🍎", "🍌", "🍌", "🍎", "🍎"}, "🍌", []string{"🍎", "🍇", "🍊"}, "", "", 1},
		{"pt04", "一二一二", "pattern", []string{"1️⃣", "2️⃣", "1️⃣", "2️⃣"}, "1️⃣", []string{"3️⃣", "4️⃣", "0️⃣"}, "", "", 1},
		{"pt05", "星星月亮", "pattern", []string{"⭐", "🌙", "⭐", "🌙"}, "⭐", []string{"☀️", "☁️", "🌈"}, "", "", 1},
		{"pt06", "笑脸重复", "pattern", []string{"😊", "😊", "😢", "😊", "😊"}, "😢", []string{"😊", "😡", "😴"}, "", "", 2},
		{"pt07", "颜色递进", "pattern", []string{"🟥", "🟧", "🟨"}, "🟩", []string{"🟦", "⬛", "⬜"}, "", "彩虹顺序下一个是？", 2},
		{"pt08", "动物成对", "pattern", []string{"🐶", "🐶", "🐱", "🐱", "🐭"}, "🐭", []string{"🐶", "🐱", "🐰"}, "", "", 2},
		{"pt09", "加减形状", "pattern", []string{"🔺", "🔺", "🔺", "⬛", "⬛"}, "⬛", []string{"🔺", "🔵", "⭐"}, "", "", 2},
		{"pt10", "数字加一", "pattern", []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣"}, "5️⃣", []string{"6️⃣", "3️⃣", "0️⃣"}, "", "接下来是几？", 2},
		{"pt11", "圆方圆方", "pattern", []string{"🔵", "⬛", "🔵", "⬛"}, "🔵", []string{"⬛", "🔺", "⭐"}, "", "", 1},
		{"pt12", "花草花草", "pattern", []string{"🌸", "🌿", "🌸", "🌿"}, "🌸", []string{"🌿", "🌳", "🍎"}, "", "", 1},
		{"pt13", "ABA 型", "pattern", []string{"🚗", "🚌", "🚗", "🚌"}, "🚗", []string{"🚕", "🚌", "🚲"}, "", "", 1},
		{"pt14", "三连循环", "pattern", []string{"🔴", "🟡", "🟢", "🔴", "🟡"}, "🟢", []string{"🔴", "🟡", "🔵"}, "", "", 2},
		{"pt15", "空心实心", "pattern", []string{"⭕", "⚫", "⭕", "⚫"}, "⭕", []string{"⚫", "⬜", "🔺"}, "", "", 2},
	}
}

func classifyItems() []logicItem {
	// classify：四个选项里选出「和其他不一样」的那个；A 是那个异类的 label/emoji
	return []logicItem{
		{"cl01", "哪个不是水果", "classify", nil, "🚗", []string{"🍎", "🍌", "🍊"}, "哪个不是水果？", "哪个不是水果？", 1},
		{"cl02", "哪个不会飞", "classify", nil, "🐕", []string{"🐦", "🦋", "✈️"}, "哪个不会飞？", "哪个不会飞？", 1},
		{"cl03", "哪个不是动物", "classify", nil, "🌳", []string{"🐱", "🐶", "🐰"}, "哪个不是动物？", "哪个不是动物？", 1},
		{"cl04", "哪个不能吃", "classify", nil, "👟", []string{"🍞", "🍎", "🧀"}, "哪个不能吃？", "哪个不能吃？", 1},
		{"cl05", "哪个不是交通工具", "classify", nil, "🏠", []string{"🚌", "🚲", "🚗"}, "哪个不是车？", "哪个不是车？", 1},
		{"cl06", "哪个不是圆形", "classify", nil, "⬛", []string{"🔵", "⚪", "🟡"}, "哪个不是圆的？", "哪个不是圆的？", 1},
		{"cl07", "哪个不是天气", "classify", nil, "🍕", []string{"☀️", "🌧️", "❄️"}, "哪个不是天气？", "哪个不是天气？", 1},
		{"cl08", "哪个不是身体部位", "classify", nil, "🎩", []string{"👀", "👂", "👃"}, "哪个不是身体上的？", "哪个不是身体上的？", 1},
		{"cl09", "哪个不是文具", "classify", nil, "🍦", []string{"✏️", "📏", "📕"}, "哪个不是学习用品？", "哪个不是学习用品？", 1},
		{"cl10", "哪个是热的", "classify", nil, "🔥", []string{"🧊", "❄️", "💧"}, "哪个是热的？", "哪个是热的？", 2},
		{"cl11", "哪个生活在水里", "classify", nil, "🐟", []string{"🐱", "🐔", "🐄"}, "哪个生活在水里？", "哪个生活在水里？", 1},
		{"cl12", "哪个是晚上", "classify", nil, "🌙", []string{"☀️", "🌅", "🌤️"}, "哪个是晚上的？", "哪个是晚上的？", 2},
	}
}

func orderItems() []logicItem {
	// order：正确项是「排好的顺序」文字描述；干扰项是错误顺序
	return []logicItem{
		{"od01", "从小到大", "order", []string{"1️⃣", "3️⃣", "2️⃣"}, "1 → 2 → 3", []string{"3 → 2 → 1", "2 → 1 → 3", "1 → 3 → 2"}, "从小到大怎么排？", "从小到大怎么排？", 1},
		{"od02", "从大到小", "order", []string{"5️⃣", "1️⃣", "3️⃣"}, "5 → 3 → 1", []string{"1 → 3 → 5", "3 → 5 → 1", "5 → 1 → 3"}, "从大到小怎么排？", "从大到小怎么排？", 1},
		{"od03", "一天顺序", "order", []string{"🌅", "☀️", "🌙"}, "早上 → 中午 → 晚上", []string{"晚上 → 早上 → 中午", "中午 → 晚上 → 早上", "早上 → 晚上 → 中午"}, "一天的正确顺序是？", "一天的正确顺序是？", 1},
		{"od04", "生长顺序", "order", []string{"🌱", "🌿", "🌳"}, "种子 → 小苗 → 大树", []string{"大树 → 小苗 → 种子", "小苗 → 种子 → 大树", "种子 → 大树 → 小苗"}, "小树长大的顺序是？", "小树长大的顺序是？", 2},
		{"od05", "穿衣顺序", "order", []string{"🧦", "👟"}, "先穿袜子再穿鞋", []string{"先穿鞋再穿袜子", "只穿鞋", "只穿袜子"}, "出门穿鞋应该？", "出门穿鞋应该怎么穿？", 1},
		{"od06", "洗手顺序", "order", []string{"💧", "🧼", "🙌"}, "打湿 → 打肥皂 → 搓洗", []string{"搓洗 → 打湿 → 打肥皂", "只冲水", "先擦干再洗"}, "洗手的正确顺序是？", "洗手的正确顺序是？", 2},
		{"od07", "字母顺序", "order", []string{"🅰️", "🅱️", "©️"}, "A → B → C", []string{"C → B → A", "B → A → C", "A → C → B"}, "字母怎么排？", "字母正确顺序是？", 2},
		{"od08", "季节顺序", "order", []string{"🌸", "☀️", "🍂", "❄️"}, "春 → 夏 → 秋 → 冬", []string{"冬 → 秋 → 夏 → 春", "夏 → 春 → 冬 → 秋", "春 → 秋 → 夏 → 冬"}, "四季的顺序是？", "四季的顺序是？", 2},
		{"od09", "身高排队", "order", []string{"👶", "👧", "👩"}, "矮 → 中 → 高", []string{"高 → 中 → 矮", "中 → 矮 → 高", "矮 → 高 → 中"}, "按身高从矮到高？", "按身高从矮到高怎么排？", 1},
		{"od10", "吃饭步骤", "order", []string{"🧼", "🍚", "🧽"}, "洗手 → 吃饭 → 收拾", []string{"吃饭 → 洗手 → 收拾", "收拾 → 吃饭 → 洗手", "洗手 → 收拾 → 吃饭"}, "吃饭前后该怎么做？", "吃饭前后该怎么做？", 2},
		{"od11", "数数顺序", "order", []string{"2️⃣", "4️⃣", "6️⃣"}, "2 → 4 → 6", []string{"6 → 4 → 2", "2 → 6 → 4", "4 → 2 → 6"}, "偶数怎么往下排？", "二四六接下来怎么排？", 2},
		{"od12", "故事顺序", "order", []string{"🥚", "🐣", "🐥"}, "蛋 → 破壳 → 小鸡", []string{"小鸡 → 蛋 → 破壳", "破壳 → 小鸡 → 蛋", "蛋 → 小鸡 → 破壳"}, "小鸡出生的顺序是？", "小鸡出生的顺序是？", 1},
	}
}

func shapeReasonItems() []logicItem {
	// 图形推理：本质也是 pattern，用几何形状
	return []logicItem{
		{"sr01", "圆圆方方", "pattern", []string{"🔵", "⬛", "🔵", "⬛"}, "🔵", []string{"🔺", "⭐", "⬛"}, "", "", 1},
		{"sr02", "三角递增", "pattern", []string{"🔺", "🔺🔺", "🔺🔺🔺"}, "🔺🔺🔺🔺", []string{"🔺", "⬛", "🔵"}, "", "三角形越来越多，下一个？", 2},
		{"sr03", "颜色形状", "pattern", []string{"🔴", "🔺", "🔴", "🔺"}, "🔴", []string{"🔵", "⬛", "⭐"}, "", "", 1},
		{"sr04", "大小圆", "pattern", []string{"⚪", "⚫", "⚪", "⚫"}, "⚪", []string{"⚫", "⬛", "🔺"}, "", "", 1},
		{"sr05", "星方星方", "pattern", []string{"⭐", "⬛", "⭐", "⬛"}, "⭐", []string{"⬛", "🔵", "❤️"}, "", "", 1},
		{"sr06", "两边对称", "pattern", []string{"🔵", "⬛", "🔵"}, "⬛", []string{"🔵", "🔺", "⭐"}, "", "中间缺哪个？", 2},
		{"sr07", "三色循环", "pattern", []string{"🟥", "🟨", "🟦", "🟥", "🟨"}, "🟦", []string{"🟥", "🟩", "⬛"}, "", "", 2},
		{"sr08", "空心实心方", "pattern", []string{"⬜", "⬛", "⬜", "⬛"}, "⬜", []string{"⬛", "🔵", "🔺"}, "", "", 1},
		{"sr09", "点点增加", "pattern", []string{"•", "••", "•••"}, "••••", []string{"•", "••", "•••••"}, "", "点子越来越多，下一个？", 2},
		{"sr10", "箭头转向", "pattern", []string{"➡️", "⬇️", "⬅️", "⬆️"}, "➡️", []string{"⬇️", "⬅️", "⬆️"}, "", "箭头转一圈，下一个？", 3},
		{"sr11", "红绿红绿", "pattern", []string{"🔴", "🟢", "🔴", "🟢"}, "🔴", []string{"🟢", "🔵", "🟡"}, "", "", 1},
		{"sr12", "方圆三角", "pattern", []string{"⬛", "🔵", "🔺", "⬛", "🔵"}, "🔺", []string{"⬛", "🔵", "⭐"}, "", "", 2},
	}
}

func diffItems() []logicItem {
	// 四个互不相同的选项，选出那个「和其他不一类」的。
	// 不能三个选项长得一样——选项去重后只剩两个，孩子也点不清。
	return []logicItem{
		{"df01", "三个水果一个梨", "classify", nil, "🍐", []string{"🍎", "🍌", "🍊"}, "哪个和其他不一样？", "哪个和其他不一样？", 1},
		{"df02", "三只动物一只狗", "classify", nil, "🐶", []string{"🐱", "🐰", "🐻"}, "哪个和其他不一样？", "哪个和其他不一样？", 1},
		{"df03", "三个笑一个哭", "classify", nil, "😢", []string{"😊", "😄", "😁"}, "哪个表情不一样？", "哪个表情不一样？", 1},
		{"df04", "三个蓝一个红", "classify", nil, "🔴", []string{"🔵", "🟦", "🔹"}, "哪个颜色不一样？", "哪个颜色不一样？", 1},
		{"df05", "三个方一个圆", "classify", nil, "🔵", []string{"⬛", "⬜", "◼️"}, "哪个形状不一样？", "哪个形状不一样？", 1},
		{"df06", "三辆车一架飞机", "classify", nil, "✈️", []string{"🚗", "🚌", "🚕"}, "哪个和其他不一样？", "哪个和其他不一样？", 1},
		{"df07", "三个太阳一个月亮", "classify", nil, "🌙", []string{"☀️", "🌞", "🌤️"}, "哪个和其他不一样？", "哪个和其他不一样？", 1},
		{"df08", "三个花一个树", "classify", nil, "🌳", []string{"🌸", "🌺", "🌹"}, "哪个和其他不一样？", "哪个和其他不一样？", 1},
		{"df09", "三个8一个9", "classify", nil, "9️⃣", []string{"8️⃣", "7️⃣", "6️⃣"}, "哪个数字不一样？", "哪个数字不一样？", 2},
		{"df10", "三个右一个左", "classify", nil, "👈", []string{"👉", "👆", "👇"}, "哪个方向不一样？", "哪个方向不一样？", 2},
		{"df11", "三个亮一个暗", "classify", nil, "🌑", []string{"🌕", "🌝", "🌞"}, "哪个和其他不一样？", "哪个和其他不一样？", 2},
	}
}
