// Package quiz 从知识点自动生成选择题。
//
// 全部是纯函数：输入知识点（标题、payload、同模块兄弟节点），输出题目规格。
// 同一个知识点每次生成的题目完全一致——随机数种子取自 kp.ID，
// 这样重复灌库不会产生内容不同但 code 相同的题。
package quiz

import (
	"encoding/json"
	"math/rand"
	"strconv"
)

// 选项固定 4 个：3 个选项猜中率太高，掌握度信号会失真；
// 6 个在 iPad 上单格过小，也超出 5 岁孩子的视觉扫描能力。
const optionCount = 4

const (
	LangZH = "zh-CN"
	LangEN = "en-US"
)

// Kp 是生成题目所需的知识点上下文。
type Kp struct {
	ID          int64
	Title       string
	Payload     string
	Difficulty  int
	SubjectCode string
	ModuleCode  string
	// Siblings 是同模块内其他知识点的标题，用作干扰项池。
	// 同模块的内容天然相近（同一组汉字、同一个话题的单词），
	// 比全局随机抽取更难靠排除法蒙对。
	Siblings []string
}

// Option 三个字段互斥：Shape 优先画矢量图形，其次 Emoji 画大图，最后回退到文字。
type Option struct {
	Label string `json:"label,omitempty"`
	Emoji string `json:"emoji,omitempty"`
	Shape string `json:"shape,omitempty"`
}

// Visual 描述题干上方的图形区，由前端按 Kind 渲染。
type Visual struct {
	Kind  string   `json:"kind,omitempty"` // count|add|sub|compare|shape|char|emoji|seq
	A     int      `json:"a,omitempty"`
	B     int      `json:"b,omitempty"`
	Emoji string   `json:"emoji,omitempty"`
	Text  string   `json:"text,omitempty"`
	Items []string `json:"items,omitempty"` // seq：规律题的 emoji 序列
}

// Speech 是朗读内容。分 Lang 是因为中英混读必须拆成两段，否则 TTS 会串味。
type Speech struct {
	Text string `json:"text,omitempty"`
	Lang string `json:"lang,omitempty"`
}

type Spec struct {
	Code        string
	Stem        string
	Options     []Option
	AnswerIndex int
	Visual      Visual
	Speech      Speech
	Difficulty  int
}

// Generate 返回该知识点的全部题目变体；不支持的学科返回 nil。
func Generate(kp Kp) []Spec {
	var specs []Spec
	switch kp.SubjectCode {
	case "math":
		specs = mathSpecs(kp)
	case "pinyin":
		specs = pinyinSpecs(kp)
	case "literacy":
		specs = literacySpecs(kp)
	case "english":
		specs = englishSpecs(kp)
	case "science":
		specs = scienceSpecs(kp)
	case "poem":
		specs = poemSpecs(kp)
	case "logic":
		specs = logicSpecs(kp)
	case "chengyu":
		specs = chengyuSpecs(kp)
	case "phrase":
		specs = phraseSpecs(kp)
	}
	for i := range specs {
		if specs[i].Difficulty == 0 {
			specs[i].Difficulty = kp.Difficulty
		}
	}
	return specs
}

func (s Spec) OptionsJSON() string { return mustJSON(s.Options) }
func (s Spec) VisualJSON() string  { return mustJSON(s.Visual) }
func (s Spec) SpeechJSON() string  { return mustJSON(s.Speech) }
func (s Spec) AnswerJSON() string  { return mustJSON(map[string]int{"index": s.AnswerIndex}) }

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func rngFor(kpID int64, variant int) *rand.Rand {
	return rand.New(rand.NewSource(kpID*1000 + int64(variant)))
}

// pickTiered 按优先级分层挑最多 n 个干扰项：先取满第一层，不够才降到下一层。
//
// 分层是干扰项质量的关键。把所有候选混在一起打乱，"2 + 5 = ?" 会挑出 "2" 这种
// 一眼就能排除的选项，孩子不用算也能做对，掌握度就测不准了。
func pickTiered(tiers [][]string, n int, reject func(string) bool, rng *rand.Rand) []string {
	out := make([]string, 0, n)
	taken := map[string]bool{}

	for _, tier := range tiers {
		if len(out) == n {
			break
		}
		shuffled := make([]string, len(tier))
		copy(shuffled, tier)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

		for _, v := range shuffled {
			if len(out) == n {
				break
			}
			if taken[v] || reject(v) {
				continue
			}
			taken[v] = true
			out = append(out, v)
		}
	}
	return out
}

func tierSize(tiers [][]string) int {
	n := 0
	for _, t := range tiers {
		n += len(t)
	}
	return n
}

// buildOptions 打乱正确项与干扰项，返回选项列表和正确项下标。
// 调用方必须保证干扰项与正确项不相等，否则下标可能落在重复项上。
func buildOptions(correct Option, distractors []Option, rng *rand.Rand) ([]Option, int) {
	opts := make([]Option, 0, len(distractors)+1)
	opts = append(opts, correct)
	opts = append(opts, distractors...)
	rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
	for i, o := range opts {
		if o == correct {
			return opts, i
		}
	}
	return opts, 0
}

// labelOptions 是纯文字选项的快捷方式，tiers 按干扰项优先级从高到低排列。
func labelOptions(correct string, tiers [][]string, rng *rand.Rand) ([]Option, int) {
	picked := pickTiered(tiers, optionCount-1, func(s string) bool { return s == correct }, rng)
	ds := make([]Option, 0, len(picked))
	for _, p := range picked {
		ds = append(ds, Option{Label: p})
	}
	return buildOptions(Option{Label: correct}, ds, rng)
}

func itoa(n int) string { return strconv.Itoa(n) }
