package quiz

import "encoding/json"

type factPayload struct {
	Kind  string   `json:"kind"`
	Q     string   `json:"q"`
	A     string   `json:"a"`
	Wrong []string `json:"wrong"`
	Emoji string   `json:"emoji"`
}

func scienceSpecs(kp Kp) []Spec {
	var p factPayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.Kind != "fact" || p.Q == "" || p.A == "" {
		return nil
	}
	if len(p.Wrong) < optionCount-1 {
		return nil
	}

	specs := make([]Spec, 0, 2)
	for i, code := range []string{"fact1", "fact2"} {
		opts, idx := labelOptions(p.A, [][]string{p.Wrong}, rngFor(kp.ID, i+1))
		sp := Spec{
			Code: code, Stem: p.Q,
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: p.Q, Lang: LangZH},
		}
		if p.Emoji != "" {
			sp.Visual = Visual{Kind: "emoji", Emoji: p.Emoji}
		}
		specs = append(specs, sp)
	}
	return specs
}
