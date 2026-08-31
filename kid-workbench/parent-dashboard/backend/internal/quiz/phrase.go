package quiz

import "encoding/json"

type phrasePayload struct {
	Kind  string   `json:"kind"`
	Zh    string   `json:"zh"`
	Wrong []string `json:"wrong"`
}

func phraseSpecs(kp Kp) []Spec {
	var p phrasePayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.Kind != "phrase" || p.Zh == "" {
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

	// 变体 1：听英语，选中文
	{
		opts, idx := labelOptions(p.Zh, [][]string{p.Wrong}, rngFor(kp.ID, 1))
		specs = append(specs, Spec{
			Code: "listen_zh", Stem: "听一听，选出中文意思",
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: kp.Title, Lang: LangEN},
		})
	}

	// 变体 2：听英语，点出英语短句
	{
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 2))
		specs = append(specs, Spec{
			Code: "listen_en", Stem: "听一听，点出这句英语",
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: kp.Title, Lang: LangEN},
		})
	}

	return specs
}
