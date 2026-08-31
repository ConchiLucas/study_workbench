package seed

import (
	"fmt"
)

type phraseItem struct {
	En    string
	Zh    string
	Wrong []string
	Diff  int
}

func phraseModules() []moduleSpec {
	topics := []struct {
		code   string
		name   string
		prefix string
		items  []phraseItem
	}{
		{"greet", "问候", "ph", []phraseItem{
			{"Good morning.", "早上好。", []string{"下午好。", "晚上好。", "晚安。"}, 1},
			{"Good afternoon.", "下午好。", []string{"早上好。", "晚安。", "再见。"}, 1},
			{"Good evening.", "晚上好。", []string{"早上好。", "下午好。", "晚安。"}, 1},
			{"Good night.", "晚安。", []string{"早上好。", "下午好。", "你好。"}, 1},
			{"Hello!", "你好！", []string{"再见。", "谢谢。", "对不起。"}, 1},
			{"How are you?", "你好吗？", []string{"你叫什么名字？", "再见。", "早上好。"}, 1},
			{"I'm fine.", "我很好。", []string{"我饿了。", "我累了。", "我不舒服。"}, 1},
			{"Nice to meet you.", "很高兴见到你。", []string{"再见。", "明天见。", "谢谢你。"}, 1},
		}},
		{"class", "课堂", "pc", []phraseItem{
			{"Sit down, please.", "请坐下。", []string{"请站起来。", "请举手。", "请安静。"}, 1},
			{"Stand up, please.", "请站起来。", []string{"请坐下。", "请打开书。", "请合上书。"}, 1},
			{"Listen to me.", "听我说。", []string{"看着我。", "跟我读。", "请安静。"}, 1},
			{"Look at me.", "看着我。", []string{"听我说。", "举手。", "坐下。"}, 1},
			{"Open your book.", "打开书。", []string{"合上书本。", "站起来。", "坐下。"}, 1},
			{"Close your book.", "合上书本。", []string{"打开书。", "举手。", "站起来。"}, 1},
			{"Raise your hand.", "举手。", []string{"坐下。", "站起来。", "安静。"}, 1},
			{"Let's begin.", "我们开始吧。", []string{"再见。", "休息吧。", "请坐下。"}, 1},
		}},
		{"daily", "日常", "pd", []phraseItem{
			{"Thank you.", "谢谢你。", []string{"不客气。", "对不起。", "再见。"}, 1},
			{"You're welcome.", "不客气。", []string{"谢谢你。", "对不起。", "再见。"}, 1},
			{"Excuse me.", "对不起/打扰一下。", []string{"谢谢你。", "再见。", "你好。"}, 1},
			{"I'm sorry.", "我很抱歉。", []string{"谢谢你。", "不客气。", "没关系。"}, 1},
			{"May I come in?", "我可以进来吗？", []string{"请坐下。", "请出去。", "请安静。"}, 1},
			{"What's your name?", "你叫什么名字？", []string{"你好吗？", "再见。", "早上好。"}, 1},
			{"My name is Lily.", "我的名字是莉莉。", []string{"你叫什么名字？", "再见。", "早上好。"}, 1},
			{"See you tomorrow.", "明天见。", []string{"晚安。", "再见。", "早上好。"}, 1},
		}},
		{"feel", "感受", "pf", []phraseItem{
			{"I like it.", "我喜欢。", []string{"我不喜欢。", "我饿了。", "我累了。"}, 1},
			{"I don't like it.", "我不喜欢。", []string{"我喜欢。", "我很好。", "我很高兴。"}, 1},
			{"I'm hungry.", "我饿了。", []string{"我渴了。", "我累了。", "我很好。"}, 1},
			{"I'm thirsty.", "我渴了。", []string{"我饿了。", "我累了。", "我很好。"}, 1},
			{"Let's play.", "我们一起玩吧。", []string{"安静。", "坐下。", "再见。"}, 1},
			{"Come here.", "过来。", []string{"等等我。", "再见。", "坐下。"}, 1},
			{"Wait for me.", "等等我。", []string{"过来。", "再见。", "开始吧。"}, 1},
			{"Be quiet.", "安静。", []string{"大声点。", "一起玩。", "站起来。"}, 1},
		}},
	}

	mods := make([]moduleSpec, 0, len(topics))
	for _, t := range topics {
		kps := make([]kpSpec, 0, len(t.items))
		for i, it := range t.items {
			diff := it.Diff
			if diff == 0 {
				diff = 1
			}
			kps = append(kps, kpSpec{
				Code: fmt.Sprintf("%s%03d", t.prefix, i+1), Title: it.En, Difficulty: diff,
				Payload: mustPayload(map[string]any{
					"kind": "phrase", "zh": it.Zh, "wrong": it.Wrong,
				}),
			})
		}
		mods = append(mods, moduleSpec{Code: t.code, Name: t.name, Kps: kps})
	}
	return mods
}
