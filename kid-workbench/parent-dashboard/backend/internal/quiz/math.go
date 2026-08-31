package quiz

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
)

type mathPayload struct {
	Kind  string `json:"kind"`
	N     int    `json:"n"`
	A     int    `json:"a"`
	B     int    `json:"b"`
	Emoji string `json:"emoji"`
}

func mathSpecs(kp Kp) []Spec {
	if kp.ModuleCode == "shape" {
		return shapeSpecs(kp)
	}
	var p mathPayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil {
		return nil
	}
	switch p.Kind {
	case "add", "sub":
		return arithmeticSpecs(kp, p)
	}
	return nil
}

// consecutiveNumberOptions 生成恰好 4 个连续整数选项（含答案），再打乱顺序。
// 例如答案 12 → 可能是 11,12,13,14（显示顺序会打乱，集合仍连续）。
func consecutiveNumberOptions(n, lo, hi int, rng *rand.Rand) ([]Option, int) {
	if hi-lo+1 < optionCount {
		return nil, 0
	}
	maxStart := hi - optionCount + 1
	var starts []int
	for s := n - optionCount + 1; s <= n; s++ {
		if s >= lo && s <= maxStart {
			starts = append(starts, s)
		}
	}
	if len(starts) == 0 {
		return nil, 0
	}
	start := starts[rng.Intn(len(starts))]
	opts := make([]Option, optionCount)
	for i := 0; i < optionCount; i++ {
		opts[i] = Option{Label: itoa(start + i)}
	}
	rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
	answer := itoa(n)
	idx := -1
	for i, o := range opts {
		if o.Label == answer {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, 0
	}
	return opts, idx
}

func arithmeticSpecs(kp Kp, p mathPayload) []Spec {
	sign, verb, story := "+", "加", "一共有几个？"
	result := p.A + p.B
	if p.Kind == "sub" {
		sign, verb, story = "-", "减", "还剩几个？"
		result = p.A - p.B
	}
	if result < 0 || result > 20 {
		return nil
	}

	visual := Visual{Kind: p.Kind, A: p.A, B: p.B, Emoji: p.Emoji}
	calcOpts, calcIdx := consecutiveNumberOptions(result, 0, 20, rngFor(kp.ID, 1))
	storyOpts, storyIdx := consecutiveNumberOptions(result, 0, 20, rngFor(kp.ID, 2))
	if len(calcOpts) != optionCount || len(storyOpts) != optionCount {
		return nil
	}

	return []Spec{
		{
			Code: "calc", Stem: fmt.Sprintf("%d %s %d = ?", p.A, sign, p.B),
			Options: calcOpts, AnswerIndex: calcIdx, Visual: visual,
			Speech: Speech{Text: fmt.Sprintf("%s%s%s等于几", cnNumber(p.A), verb, cnNumber(p.B)), Lang: LangZH},
		},
		{
			// 同一道算式的"看图版"：不给算式，只给实物，练的是数量感而不是符号运算。
			Code: "story", Stem: story,
			Options: storyOpts, AnswerIndex: storyIdx, Visual: visual,
			Speech: Speech{Text: story, Lang: LangZH},
		},
	}
}

var cnDigits = []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九", "十"}

func cnNumber(n int) string {
	if n < 0 {
		return itoa(n)
	}
	if n < len(cnDigits) {
		return cnDigits[n]
	}
	if n < 20 {
		return "十" + cnDigits[n-10]
	}
	if n == 20 {
		return "二十"
	}
	return itoa(n)
}

// 图形题用矢量图形而不是 emoji：Unicode 里没有可靠的长方形、椭圆、梯形字符，
// 各系统字体表现差异很大，前端画 SVG 更可控。
var shapeKeys = map[string]string{
	"圆形": "circle", "正方形": "square", "长方形": "rect", "三角形": "triangle",
	"椭圆形": "oval", "梯形": "trapezoid", "菱形": "rhombus", "五角星": "star",
}

func shapeSpecs(kp Kp) []Spec {
	key, ok := shapeKeys[kp.Title]
	if !ok {
		return nil
	}

	var names []string
	for name := range shapeKeys {
		names = append(names, name)
	}
	// map 遍历顺序随机，先排序保证生成结果稳定。
	sort.Strings(names)

	findOpts, findIdx := shapeOptions(kp.Title, names, rngFor(kp.ID, 1))
	nameOpts, nameIdx := labelOptions(kp.Title, [][]string{names}, rngFor(kp.ID, 2))

	return []Spec{
		{
			Code: "find", Stem: "点出" + kp.Title,
			Options: findOpts, AnswerIndex: findIdx,
			Speech: Speech{Text: "点出" + kp.Title, Lang: LangZH},
		},
		{
			Code: "name", Stem: "这是什么形状？",
			Options: nameOpts, AnswerIndex: nameIdx,
			Visual: Visual{Kind: "shape", Text: key},
			Speech: Speech{Text: "这是什么形状", Lang: LangZH},
		},
	}
}

func shapeOptions(correct string, names []string, rng *rand.Rand) ([]Option, int) {
	picked := pickTiered([][]string{names}, optionCount-1,
		func(s string) bool { return s == correct }, rng)
	ds := make([]Option, 0, len(picked))
	for _, p := range picked {
		ds = append(ds, Option{Shape: shapeKeys[p]})
	}
	return buildOptions(Option{Shape: shapeKeys[correct]}, ds, rng)
}
