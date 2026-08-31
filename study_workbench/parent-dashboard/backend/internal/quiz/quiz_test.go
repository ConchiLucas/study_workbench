package quiz

import (
	"sort"
	"strconv"
	"testing"
)

func optionLabels(opts []Option) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.Label)
	}
	return out
}

func find(specs []Spec, code string) (Spec, bool) {
	for _, s := range specs {
		if s.Code == code {
			return s, true
		}
	}
	return Spec{}, false
}

// 所有题目都必须满足的基本契约：4 个选项、正确项下标在范围内、选项互不相同。
func assertWellFormed(t *testing.T, specs []Spec) {
	t.Helper()
	if len(specs) == 0 {
		t.Fatal("没有生成任何题目")
	}
	seenCode := map[string]bool{}
	for _, s := range specs {
		if seenCode[s.Code] {
			t.Errorf("变体 code 重复: %s", s.Code)
		}
		seenCode[s.Code] = true

		if len(s.Options) != optionCount {
			t.Errorf("%s: 选项数 %d，期望 %d", s.Code, len(s.Options), optionCount)
		}
		if s.AnswerIndex < 0 || s.AnswerIndex >= len(s.Options) {
			t.Errorf("%s: 正确项下标 %d 越界", s.Code, s.AnswerIndex)
		}
		if s.Stem == "" {
			t.Errorf("%s: 题干为空", s.Code)
		}
		if s.Difficulty < 1 || s.Difficulty > 3 {
			t.Errorf("%s: 难度 %d 超出 1~3", s.Code, s.Difficulty)
		}
		seen := map[Option]bool{}
		for _, o := range s.Options {
			if seen[o] {
				t.Errorf("%s: 选项重复 %+v", s.Code, o)
			}
			seen[o] = true
			if o.Label == "" && o.Emoji == "" && o.Shape == "" {
				t.Errorf("%s: 存在空选项", s.Code)
			}
		}
	}
}

func TestArithmeticAnswerIsCorrectAndDistractorsAreNear(t *testing.T) {
	kp := Kp{
		ID: 63, Title: "2+5", Payload: `{"kind":"add","a":2,"b":5,"emoji":"🍎"}`,
		Difficulty: 2, SubjectCode: "math", ModuleCode: "add10",
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	calc, ok := find(specs, "calc")
	if !ok {
		t.Fatal("缺少 calc 变体")
	}
	if calc.Stem != "2 + 5 = ?" {
		t.Errorf("题干 = %q", calc.Stem)
	}
	if got := calc.Options[calc.AnswerIndex].Label; got != "7" {
		t.Errorf("正确项 = %q，期望 7", got)
	}
	if calc.Speech.Text != "二加五等于几" {
		t.Errorf("朗读文本 = %q", calc.Speech.Text)
	}

	for _, s := range specs {
		assertConsecutiveOptions(t, s, 7)
	}
}

func TestArithmeticFallsBackToFartherDistractorsAtRangeEdge(t *testing.T) {
	// 结果是 0 时，连续窗口只能是 0,1,2,3。
	kp := Kp{
		ID: 91, Title: "1-1", Payload: `{"kind":"sub","a":1,"b":1,"emoji":"🍓"}`,
		Difficulty: 1, SubjectCode: "math", ModuleCode: "sub10",
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	calc, _ := find(specs, "calc")
	if got := calc.Options[calc.AnswerIndex].Label; got != "0" {
		t.Errorf("正确项 = %q，期望 0", got)
	}
	assertConsecutiveOptions(t, calc, 0)
	labels := optionLabels(calc.Options)
	sort.Strings(labels)
	want := []string{"0", "1", "2", "3"}
	if len(labels) != 4 || labels[0] != want[0] || labels[1] != want[1] || labels[2] != want[2] || labels[3] != want[3] {
		t.Errorf("边缘答案选项 = %v，期望 %v", labels, want)
	}
}

func TestSubtractionUsesDifferenceAsAnswer(t *testing.T) {
	kp := Kp{
		ID: 90, Title: "9-4", Payload: `{"kind":"sub","a":9,"b":4,"emoji":"🍓"}`,
		Difficulty: 3, SubjectCode: "math", ModuleCode: "sub10",
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	calc, _ := find(specs, "calc")
	if got := calc.Options[calc.AnswerIndex].Label; got != "5" {
		t.Errorf("正确项 = %q，期望 5", got)
	}
	if calc.Visual.Kind != "sub" || calc.Visual.Emoji != "🍓" {
		t.Errorf("图形区 = %+v", calc.Visual)
	}
	assertConsecutiveOptions(t, calc, 5)
}

func assertConsecutiveOptions(t *testing.T, s Spec, answer int) {
	t.Helper()
	if len(s.Options) != 4 {
		t.Fatalf("选项数 = %d", len(s.Options))
	}
	nums := make([]int, 0, 4)
	for _, o := range s.Options {
		n, err := strconv.Atoi(o.Label)
		if err != nil {
			t.Fatalf("选项不是整数: %q", o.Label)
		}
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for i := 1; i < len(nums); i++ {
		if nums[i] != nums[i-1]+1 {
			t.Fatalf("选项必须是连续 4 个数，got %v", nums)
		}
	}
	if nums[0] > answer || nums[3] < answer {
		t.Fatalf("连续窗口 %v 未包含答案 %d", nums, answer)
	}
	if got := s.Options[s.AnswerIndex].Label; got != itoa(answer) {
		t.Fatalf("AnswerIndex 指向 %q，期望 %d", got, answer)
	}
}

func TestShapeOptionsUseVectorShapesNotEmoji(t *testing.T) {
	kp := Kp{ID: 300, Title: "梯形", SubjectCode: "math", ModuleCode: "shape", Difficulty: 3}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	findSpec, ok := find(specs, "find")
	if !ok {
		t.Fatal("缺少 find 变体")
	}
	if got := findSpec.Options[findSpec.AnswerIndex].Shape; got != "trapezoid" {
		t.Errorf("正确项 shape = %q", got)
	}
	for _, o := range findSpec.Options {
		if o.Shape == "" {
			t.Errorf("图形题选项应该都是矢量图形，得到 %+v", o)
		}
	}

	nameSpec, _ := find(specs, "name")
	if got := nameSpec.Options[nameSpec.AnswerIndex].Label; got != "梯形" {
		t.Errorf("正确项 = %q", got)
	}
}

func TestLiteracyExcludesHomophoneDistractors(t *testing.T) {
	// 「他」和「她」同音，听读音无法区分，必须不出现在同一道题里。
	group := []string{"你", "他", "她", "们", "的", "了", "不", "在", "有", "是"}
	kp := Kp{
		ID: 402, Title: "他", SubjectCode: "literacy", ModuleCode: "g7",
		Difficulty: 2, Siblings: group,
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	for _, s := range specs {
		if got := s.Options[s.AnswerIndex].Label; got != "他" {
			t.Errorf("%s: 正确项 = %q", s.Code, got)
		}
		for _, o := range s.Options {
			if o.Label == "她" {
				t.Errorf("%s: 同音字「她」出现在选项里", s.Code)
			}
		}
	}
	if _, ok := find(specs, "listen_glyph"); ok {
		t.Fatal("不应再生成 listen_glyph")
	}
	if _, ok := find(specs, "glyph_sense"); !ok {
		t.Fatal("缺少 glyph_sense")
	}
	if _, ok := find(specs, "sense_char"); !ok {
		t.Fatal("缺少 sense_char")
	}
}

func TestLiteracyUnknownCharIsSkipped(t *testing.T) {
	kp := Kp{
		ID: 999, Title: "龘", SubjectCode: "literacy", ModuleCode: "g1",
		Difficulty: 1, Siblings: []string{"一", "二", "三", "四"},
	}
	if specs := Generate(kp); len(specs) != 0 {
		t.Errorf("没有读音表的字不该出题，却生成了 %d 道", len(specs))
	}
}

func TestPinyinSpeaksHanziNotLetterName(t *testing.T) {
	kp := Kp{
		ID: 500, Title: "b", SubjectCode: "pinyin", ModuleCode: "shengmu",
		Difficulty: 1, Siblings: []string{"b", "p", "m", "f", "d", "t"},
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	solo, ok := find(specs, "listen")
	if !ok {
		t.Fatal("缺少 listen 变体")
	}
	if solo.Speech.Text != "波" {
		t.Errorf("单读音应该用汉字承载，得到 %q", solo.Speech.Text)
	}

	inword, ok := find(specs, "inword")
	if !ok {
		t.Fatal("缺少 inword 变体")
	}
	if inword.Speech.Text != "爸" || inword.Visual.Text != "爸" {
		t.Errorf("例字变体 = %+v / %+v", inword.Speech, inword.Visual)
	}

	// b 的易混组 d/p/q 刚好 3 个，干扰项必须全部来自这里，
	// 不能因为同模块还有 20 个声母就随便抓。
	for _, s := range specs {
		for _, o := range s.Options {
			if !contains([]string{"b", "d", "p", "q"}, o.Label) {
				t.Errorf("%s: 干扰项 %q 不在形近组 b/d/p/q 内", s.Code, o.Label)
			}
		}
	}
}

func TestPinyinEngHasOnlyExampleWordVariant(t *testing.T) {
	// eng 没有可靠的单读汉字，只应出例字变体。
	kp := Kp{
		ID: 545, Title: "eng", SubjectCode: "pinyin", ModuleCode: "yunmu",
		Difficulty: 2, Siblings: []string{"ang", "eng", "an", "en", "in", "un"},
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	if _, ok := find(specs, "listen"); ok {
		t.Error("eng 不该有单读音变体")
	}
	inword, ok := find(specs, "inword")
	if !ok {
		t.Fatal("缺少 inword 变体")
	}
	if inword.Speech.Text != "灯" {
		t.Errorf("例字 = %q，期望 灯", inword.Speech.Text)
	}
	// eng 的易混组是 ang/eng/an/en，前后鼻音混淆才是这里要练的点。
	for _, o := range inword.Options {
		if !contains([]string{"eng", "ang", "an", "en"}, o.Label) {
			t.Errorf("干扰项 %q 不在前后鼻音易混组内", o.Label)
		}
	}
}

func TestEnglishReadsWordInEnglishAndPicksSameTopicDistractors(t *testing.T) {
	topic := []string{"cat", "dog", "bird", "fish", "rabbit", "tiger", "lion", "bear", "monkey", "panda"}
	kp := Kp{
		ID: 600, Title: "cat", SubjectCode: "english", ModuleCode: "animals",
		Difficulty: 1, Siblings: topic,
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	listen, _ := find(specs, "listen")
	if listen.Speech.Lang != LangEN || listen.Speech.Text != "cat" {
		t.Errorf("朗读 = %+v，英语单词必须用 en-US", listen.Speech)
	}
	for _, o := range listen.Options {
		if !contains(topic, o.Label) {
			t.Errorf("干扰项 %q 不在同话题内", o.Label)
		}
	}

	picture, ok := find(specs, "picture")
	if !ok {
		t.Fatal("动物话题的单词都有 emoji，应该出看图题")
	}
	if got := picture.Options[picture.AnswerIndex].Emoji; got != "🐱" {
		t.Errorf("正确项 emoji = %q", got)
	}
}

func TestEnglishSkipsPictureVariantWithoutEnoughEmoji(t *testing.T) {
	topic := []string{"hello", "hi", "bye", "thanks", "please", "sorry", "yes", "no", "ok", "welcome"}
	kp := Kp{
		ID: 700, Title: "hello", SubjectCode: "english", ModuleCode: "greetings",
		Difficulty: 3, Siblings: topic,
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)

	if _, ok := find(specs, "picture"); ok {
		t.Error("问候语没有对应 emoji，不该出看图题")
	}
	if _, ok := find(specs, "listen"); !ok {
		t.Error("仍然应该出听音选词题")
	}
}

func TestGenerationIsDeterministic(t *testing.T) {
	kp := Kp{
		ID: 63, Title: "2+5", Payload: `{"kind":"add","a":2,"b":5,"emoji":"🍎"}`,
		Difficulty: 2, SubjectCode: "math", ModuleCode: "add10",
	}
	first, second := Generate(kp), Generate(kp)
	if len(first) != len(second) {
		t.Fatalf("题目数不稳定: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].OptionsJSON() != second[i].OptionsJSON() ||
			first[i].AnswerIndex != second[i].AnswerIndex {
			t.Errorf("第 %d 道题两次生成结果不同，重复灌库会产生脏数据", i)
		}
	}
}

func TestUnsupportedSubjectProducesNothing(t *testing.T) {
	kp := Kp{ID: 800, Title: "第1关", SubjectCode: "game", ModuleCode: "levels", Difficulty: 2}
	if specs := Generate(kp); len(specs) != 0 {
		t.Errorf("游戏科暂不支持出题，却生成了 %d 道", len(specs))
	}
}

func TestScienceFactQuestion(t *testing.T) {
	kp := Kp{
		ID: 900, Title: "冬眠", SubjectCode: "science", ModuleCode: "animal", Difficulty: 1,
		Payload: `{"kind":"fact","q":"冬天躲起来睡觉过冬的是？","a":"熊","wrong":["燕子","蝴蝶","鸭子"],"emoji":"🐻"}`,
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)
	s, ok := find(specs, "fact1")
	if !ok {
		t.Fatal("缺少 fact1")
	}
	if s.Options[s.AnswerIndex].Label != "熊" {
		t.Errorf("正确项 = %q", s.Options[s.AnswerIndex].Label)
	}
	if s.Visual.Emoji != "🐻" {
		t.Errorf("visual emoji = %q", s.Visual.Emoji)
	}
}

func TestPoemTitleFromFirstLine(t *testing.T) {
	siblings := []string{"静夜思", "春晓", "咏鹅", "悯农", "登鹳雀楼"}
	kp := Kp{
		ID: 901, Title: "静夜思", SubjectCode: "poem", ModuleCode: "poem50", Difficulty: 1,
		Siblings: siblings,
		Payload:  `{"kind":"poem","author":"李白","line1":"床前明月光","line2":"疑是地上霜"}`,
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)
	title, ok := find(specs, "title")
	if !ok {
		t.Fatal("缺少 title 变体")
	}
	if title.Options[title.AnswerIndex].Label != "静夜思" {
		t.Errorf("正确项 = %q", title.Options[title.AnswerIndex].Label)
	}
	if title.Visual.Text != "床前明月光" {
		t.Errorf("visual = %q", title.Visual.Text)
	}
	if _, ok := find(specs, "nextline"); !ok {
		t.Error("应该有下一句变体")
	}
}

func TestLogicPatternSequence(t *testing.T) {
	kp := Kp{
		ID: 902, Title: "红蓝交替", SubjectCode: "logic", ModuleCode: "pattern", Difficulty: 1,
		Payload: `{"kind":"pattern","seq":["🔴","🔵","🔴","🔵"],"a":"🔴","wrong":["🟢","🟡","⬛"],"prompt":"下一个是哪个？","speech":"下一个是哪个？"}`,
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)
	s, _ := find(specs, "pick1")
	if got := s.Options[s.AnswerIndex].Emoji; got != "🔴" {
		t.Errorf("正确项 emoji = %q", got)
	}
	if len(s.Visual.Items) != 4 {
		t.Errorf("序列长度 = %d", len(s.Visual.Items))
	}
}

func TestChengyuMeaningAndPick(t *testing.T) {
	siblings := []string{"一心一意", "二话不说", "三心二意", "五颜六色", "七上八下", "十全十美", "一毛不拔", "一举两得"}
	kp := Kp{
		ID: 910, Title: "一心一意", SubjectCode: "chengyu", ModuleCode: "daily", Difficulty: 1,
		Siblings: siblings,
		Payload:  `{"kind":"chengyu","pinyin":"yì xīn yì yì","meaning":"集中精神，做事专心","example":"做作业要一心一意。","wrong":["三心二意","慢慢来","随便玩玩"]}`,
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)
	meaning, ok := find(specs, "meaning")
	if !ok {
		t.Fatal("缺少 meaning 变体")
	}
	if meaning.Options[meaning.AnswerIndex].Label != "集中精神，做事专心" {
		t.Errorf("释义 = %q", meaning.Options[meaning.AnswerIndex].Label)
	}
	pick, ok := find(specs, "pick")
	if !ok {
		t.Fatal("缺少 pick 变体")
	}
	if pick.Options[pick.AnswerIndex].Label != "一心一意" {
		t.Errorf("成语 = %q", pick.Options[pick.AnswerIndex].Label)
	}
}

func TestPhraseListenZhAndEn(t *testing.T) {
	siblings := []string{"Good morning.", "Good afternoon.", "Good evening.", "Good night.", "Hello!", "How are you?", "I'm fine.", "Nice to meet you."}
	kp := Kp{
		ID: 920, Title: "Good morning.", SubjectCode: "phrase", ModuleCode: "greet", Difficulty: 1,
		Siblings: siblings,
		Payload:  `{"kind":"phrase","zh":"早上好。","wrong":["下午好。","晚上好。","晚安。"]}`,
	}
	specs := Generate(kp)
	assertWellFormed(t, specs)
	zh, ok := find(specs, "listen_zh")
	if !ok {
		t.Fatal("缺少 listen_zh 变体")
	}
	if zh.Options[zh.AnswerIndex].Label != "早上好。" {
		t.Errorf("中文 = %q", zh.Options[zh.AnswerIndex].Label)
	}
	if zh.Speech.Lang != LangEN {
		t.Errorf("speech lang = %q", zh.Speech.Lang)
	}
	en, ok := find(specs, "listen_en")
	if !ok {
		t.Fatal("缺少 listen_en 变体")
	}
	if en.Options[en.AnswerIndex].Label != "Good morning." {
		t.Errorf("英文 = %q", en.Options[en.AnswerIndex].Label)
	}
}

func TestAnswerJSONCarriesIndex(t *testing.T) {
	s := Spec{AnswerIndex: 2}
	if got := s.AnswerJSON(); got != `{"index":2}` {
		t.Errorf("AnswerJSON = %q", got)
	}
}
