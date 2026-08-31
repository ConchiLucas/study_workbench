package quiz

import "encoding/json"

type logicPayload struct {
	Kind   string   `json:"kind"`
	Seq    []string `json:"seq"`
	A      string   `json:"a"`
	Wrong  []string `json:"wrong"`
	Prompt string   `json:"prompt"`
	Speech string   `json:"speech"`
}

func logicSpecs(kp Kp) []Spec {
	var p logicPayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.A == "" {
		return nil
	}
	if len(p.Wrong) < optionCount-1 {
		return nil
	}

	prompt := p.Prompt
	if prompt == "" {
		prompt = "选一选"
	}
	speech := p.Speech
	if speech == "" {
		speech = prompt
	}

	// emoji 选项：答案和干扰项都是单个 emoji 时，用 Emoji 字段让按钮更大。
	useEmoji := looksLikeEmoji(p.A)
	for _, w := range p.Wrong {
		if !looksLikeEmoji(w) {
			useEmoji = false
			break
		}
	}

	specs := make([]Spec, 0, 2)
	for i, code := range []string{"pick1", "pick2"} {
		rng := rngFor(kp.ID, i+1)
		var opts []Option
		var idx int
		if useEmoji {
			picked := pickTiered([][]string{p.Wrong}, optionCount-1, func(s string) bool { return s == p.A }, rng)
			ds := make([]Option, 0, len(picked))
			for _, w := range picked {
				ds = append(ds, Option{Emoji: w})
			}
			opts, idx = buildOptions(Option{Emoji: p.A}, ds, rng)
		} else {
			opts, idx = labelOptions(p.A, [][]string{p.Wrong}, rng)
		}

		sp := Spec{
			Code: code, Stem: prompt,
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: speech, Lang: LangZH},
		}
		if len(p.Seq) > 0 {
			sp.Visual = Visual{Kind: "seq", Items: p.Seq}
		}
		specs = append(specs, sp)
	}
	return specs
}

// 粗判：不含常见汉字/字母数字、长度较短的当成 emoji 选项。
// 排序题的「1 → 2 → 3」会走文字分支。
func looksLikeEmoji(s string) bool {
	if s == "" || len([]rune(s)) > 4 {
		return false
	}
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return false
		}
		if r >= 0x4e00 && r <= 0x9fff {
			return false
		}
	}
	return true
}
