package quiz

import "encoding/json"

type chengyuPayload struct {
	Kind    string   `json:"kind"`
	Pinyin  string   `json:"pinyin"`
	Meaning string   `json:"meaning"`
	Example string   `json:"example"`
	Wrong   []string `json:"wrong"`
}

func chengyuSpecs(kp Kp) []Spec {
	var p chengyuPayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.Kind != "chengyu" || p.Meaning == "" {
		return nil
	}
	if len(p.Wrong) < optionCount-1 {
		return nil
	}

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

	// 变体 1：听成语，选释义
	{
		opts, idx := labelOptions(p.Meaning, [][]string{p.Wrong}, rngFor(kp.ID, 1))
		specs = append(specs, Spec{
			Code: "meaning", Stem: "这个成语是什么意思？",
			Options: opts, AnswerIndex: idx,
			Visual: Visual{Kind: "char", Text: kp.Title},
			Speech: Speech{Text: kp.Title, Lang: LangZH},
		})
	}

	// 变体 2：听成语，点出成语
	{
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 2))
		specs = append(specs, Spec{
			Code: "pick", Stem: "听一听，点出这个成语",
			Options: opts, AnswerIndex: idx,
			Visual: Visual{Kind: "char", Text: kp.Title},
			Speech: Speech{Text: kp.Title, Lang: LangZH},
		})
	}

	return specs
}
