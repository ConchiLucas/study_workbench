package quiz

import "encoding/json"

type poemPayload struct {
	Kind   string `json:"kind"`
	Author string `json:"author"`
	Line1  string `json:"line1"`
	Line2  string `json:"line2"`
}

func poemSpecs(kp Kp) []Spec {
	var p poemPayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.Kind != "poem" || p.Line1 == "" {
		return nil
	}

	// 干扰项：同模块其他诗名。诗名互不相同，排除法比瞎蒙难一点。
	pool := make([]string, 0, len(kp.Siblings))
	for _, s := range kp.Siblings {
		if s != kp.Title {
			pool = append(pool, s)
		}
	}
	if len(pool) < optionCount-1 {
		return nil
	}

	specs := make([]Spec, 0, 2)

	// 变体 1：听首句，点诗名
	{
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 1))
		specs = append(specs, Spec{
			Code: "title", Stem: "这是哪首诗？",
			Options: opts, AnswerIndex: idx,
			Visual: Visual{Kind: "char", Text: p.Line1},
			Speech: Speech{Text: p.Line1, Lang: LangZH},
		})
	}

	// 变体 2：听「上一句」，选下一句（干扰项从其他诗的首句里抽）
	if p.Line2 != "" {
		linePool := make([]string, 0, len(kp.Siblings))
		// 同模块其他知识点的 line1 需要从标题推不出来——用常见错配句作后备。
		// 干扰项优先用其他诗名对应的「听起来像」的短句不够，这里用固定近邻池 + 诗名池不够时的兜底。
		// 实际：从 siblings 不够表达诗句，所以用 line2 的形近干扰——换几句常见启蒙诗首句。
		linePool = poemLineDistractors(p.Line1, p.Line2)
		if len(linePool) >= optionCount-1 {
			opts, idx := labelOptions(p.Line2, [][]string{linePool}, rngFor(kp.ID, 2))
			specs = append(specs, Spec{
				Code: "nextline", Stem: "下一句是？",
				Options: opts, AnswerIndex: idx,
				Visual: Visual{Kind: "char", Text: p.Line1},
				Speech: Speech{Text: p.Line1 + "，下一句是什么", Lang: LangZH},
			})
		}
	}

	return specs
}

// 启蒙诗里高频的「下一句」干扰项。故意挑孩子可能听混的短句，
// 而不是随机汉字——否则蒙对率会虚高。
func poemLineDistractors(line1, correct string) []string {
	pool := []string{
		"疑是地上霜", "处处闻啼鸟", "曲项向天歌", "汗滴禾下土",
		"黄河入海流", "万径人踪灭", "言师采药去", "歌声振林樾",
		"树阴照水爱晴柔", "万条垂下绿丝绦", "一岁一枯荣", "呼作白玉盘",
		"忽闻岸上踏歌声", "一行白鹭上青天", "江枫渔火对愁眠", "白云生处有人家",
		"路上行人欲断魂", "游子身上衣", "春来发几枝", "莲叶何田田",
		"凌寒独自开", "但闻人语响", "手可摘星辰", "偷采白莲回",
	}
	out := make([]string, 0, len(pool))
	for _, s := range pool {
		if s != correct && s != line1 {
			out = append(out, s)
		}
	}
	return out
}
