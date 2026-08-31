package seed

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/conchi/study-workbench/internal/model"
)

type kpSpec struct {
	Code       string
	Title      string
	Payload    string
	Difficulty int
}

type moduleSpec struct {
	Code string
	Name string
	Kps  []kpSpec
}

type subjectSpec struct {
	Code string
	Name string
	Icon string
	// QuizEnabled 决定该学科是否参与自动出题与每日答题计划。
	QuizEnabled bool
	Modules     []moduleSpec
}

func Catalog(gdb *gorm.DB) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		if err := ensureDefaultFamily(tx); err != nil {
			return err
		}
		catalog := buildCatalog()
		for si, s := range catalog {
			subject := model.Subject{
				Code: s.Code, Name: s.Name, Icon: s.Icon,
				OrderNo: si, QuizEnabled: s.QuizEnabled,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "code"}},
				DoUpdates: clause.AssignmentColumns([]string{"name", "icon", "order_no", "quiz_enabled"}),
			}).Create(&subject).Error; err != nil {
				return err
			}
			if subject.ID == 0 {
				if err := tx.Where("code = ?", s.Code).First(&subject).Error; err != nil {
					return err
				}
			}

			for mi, m := range s.Modules {
				mod := model.Module{SubjectID: subject.ID, Code: m.Code, Name: m.Name, OrderNo: mi}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "subject_id"}, {Name: "code"}},
					DoUpdates: clause.AssignmentColumns([]string{"name", "order_no"}),
				}).Create(&mod).Error; err != nil {
					return err
				}
				if mod.ID == 0 {
					if err := tx.Where("subject_id = ? AND code = ?", subject.ID, m.Code).
						First(&mod).Error; err != nil {
						return err
					}
				}

				for ki, k := range m.Kps {
					payload := k.Payload
					if payload == "" {
						payload = "{}"
					}
					diff := k.Difficulty
					if diff == 0 {
						diff = 1
					}
					kp := model.KnowledgePoint{
						ModuleID: mod.ID, Code: k.Code, Title: k.Title,
						Payload: payload, Difficulty: diff, OrderNo: ki,
					}
					if err := tx.Clauses(clause.OnConflict{
						Columns:   []clause.Column{{Name: "module_id"}, {Name: "code"}},
						DoUpdates: clause.AssignmentColumns([]string{"title", "payload", "difficulty", "order_no"}),
					}).Create(&kp).Error; err != nil {
						return err
					}
				}
			}

			// 目录缩减时清掉已移除的模块/知识点（否则认数字等会残留在矩阵里）。
			if err := pruneSubjectToCatalog(tx, subject.ID, s.Modules); err != nil {
				return err
			}
		}
		return ensureDefaultRewards(tx)
	})
}

func pruneSubjectToCatalog(tx *gorm.DB, subjectID int64, modules []moduleSpec) error {
	keepMod := map[string]map[string]struct{}{}
	for _, m := range modules {
		codes := make(map[string]struct{}, len(m.Kps))
		for _, k := range m.Kps {
			codes[k.Code] = struct{}{}
		}
		keepMod[m.Code] = codes
	}

	var mods []model.Module
	if err := tx.Where("subject_id = ?", subjectID).Find(&mods).Error; err != nil {
		return err
	}
	for _, mod := range mods {
		keepKps, ok := keepMod[mod.Code]
		if !ok {
			if err := deleteModuleCascade(tx, mod.ID); err != nil {
				return err
			}
			continue
		}
		var kps []model.KnowledgePoint
		if err := tx.Where("module_id = ?", mod.ID).Find(&kps).Error; err != nil {
			return err
		}
		for _, kp := range kps {
			if _, keep := keepKps[kp.Code]; keep {
				continue
			}
			if err := deleteKpCascade(tx, kp.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteModuleCascade(tx *gorm.DB, moduleID int64) error {
	var kpIDs []int64
	if err := tx.Model(&model.KnowledgePoint{}).Where("module_id = ?", moduleID).
		Pluck("id", &kpIDs).Error; err != nil {
		return err
	}
	for _, id := range kpIDs {
		if err := deleteKpCascade(tx, id); err != nil {
			return err
		}
	}
	return tx.Where("id = ?", moduleID).Delete(&model.Module{}).Error
}

func deleteKpCascade(tx *gorm.DB, kpID int64) error {
	// plan_items → questions / attempts / mastery，按依赖顺序删。
	if err := tx.Exec(`DELETE FROM plan_items WHERE kp_id = ?`, kpID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM questions WHERE kp_id = ?`, kpID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM attempts WHERE kp_id = ?`, kpID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM mastery_states WHERE kp_id = ?`, kpID).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", kpID).Delete(&model.KnowledgePoint{}).Error
}

func ensureDefaultFamily(tx *gorm.DB) error {
	user := model.User{ID: 1, Phone: "13800000000", Nickname: "妈妈", CreatedAt: time.Now()}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"nickname"}),
	}).Create(&user).Error; err != nil {
		return err
	}
	child := model.Child{ID: 1, Name: "卢沁一", Grade: "大班", CreatedAt: time.Now()}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "grade"}),
	}).Create(&child).Error; err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO parent_child (user_id, child_id, relation, role)
		VALUES (1, 1, '妈妈', 'owner')
		ON CONFLICT (user_id, child_id) DO NOTHING`).Error
}

func ensureDefaultRewards(tx *gorm.DB) error {
	rewards := []struct {
		name  string
		cost  int
		stock int
	}{
		{"看动画片 20 分钟", 5, 10},
		{"去公园玩", 15, 5},
		{"买一本新绘本", 30, 3},
	}
	for _, r := range rewards {
		var n int64
		_ = tx.Raw(`SELECT COUNT(1) FROM rewards WHERE child_id=1 AND name=?`, r.name).Scan(&n)
		if n == 0 {
			if err := tx.Exec(`INSERT INTO rewards (child_id, name, cost, stock) VALUES (1, ?, ?, ?)`,
				r.name, r.cost, r.stock).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// 游戏的知识点仍是占位；科普/古诗/逻辑已补真实内容并可出题。
func buildCatalog() []subjectSpec {
	return []subjectSpec{
		{Code: "literacy", Name: "识字", Icon: "📖", QuizEnabled: true, Modules: literacyModules()},
		{Code: "pinyin", Name: "拼音", Icon: "🔤", QuizEnabled: true, Modules: pinyinModules()},
		{Code: "math", Name: "算术", Icon: "🔢", QuizEnabled: true, Modules: mathModules()},
		{Code: "english", Name: "英语", Icon: "🌍", QuizEnabled: true, Modules: englishModules()},
		{Code: "science", Name: "科普", Icon: "🌱", QuizEnabled: true, Modules: scienceModules()},
		{Code: "poem", Name: "古诗", Icon: "🌸", QuizEnabled: true, Modules: poemModules()},
		{Code: "logic", Name: "逻辑", Icon: "🧩", QuizEnabled: true, Modules: logicModules()},
		{Code: "chengyu", Name: "成语", Icon: "📜", QuizEnabled: true, Modules: chengyuModules()},
		{Code: "phrase", Name: "英语短句", Icon: "💬", QuizEnabled: true, Modules: phraseModules()},
		{Code: "game", Name: "游戏", Icon: "🎮", Modules: gameModules()},
	}
}

func genKps(prefix string, items []string, difficulty int) []kpSpec {
	out := make([]kpSpec, 0, len(items))
	for i, it := range items {
		out = append(out, kpSpec{
			Code:       fmt.Sprintf("%s%03d", prefix, i+1),
			Title:      it,
			Difficulty: difficulty,
		})
	}
	return out
}

func chunk(items []string, size int) [][]string {
	var out [][]string
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		out = append(out, items[i:end])
	}
	return out
}

func mathModules() []moduleSpec {
	// 20 以内加法（含 10 以内）：a+b ≤ 20
	var adds []kpSpec
	for a := 1; a <= 19; a++ {
		for b := 1; a+b <= 20; b++ {
			adds = append(adds, kpSpec{
				Code: fmt.Sprintf("%dp%d", a, b), Title: fmt.Sprintf("%d+%d", a, b),
				Payload:    fmt.Sprintf(`{"kind":"add","a":%d,"b":%d,"emoji":"🍎"}`, a, b),
				Difficulty: difficultyBySum(a + b),
			})
		}
	}

	// 20 以内减法（含 10 以内）：被减数 ≤ 20，差 ≥ 0
	var subs []kpSpec
	for a := 1; a <= 20; a++ {
		for b := 1; b <= a; b++ {
			subs = append(subs, kpSpec{
				Code: fmt.Sprintf("%dm%d", a, b), Title: fmt.Sprintf("%d-%d", a, b),
				Payload:    fmt.Sprintf(`{"kind":"sub","a":%d,"b":%d,"emoji":"🍓"}`, a, b),
				Difficulty: difficultyBySum(a),
			})
		}
	}

	shapes := []kpSpec{
		{Code: "s1", Title: "圆形", Difficulty: 1}, {Code: "s2", Title: "正方形", Difficulty: 1},
		{Code: "s3", Title: "长方形", Difficulty: 2}, {Code: "s4", Title: "三角形", Difficulty: 1},
		{Code: "s5", Title: "椭圆形", Difficulty: 2}, {Code: "s6", Title: "梯形", Difficulty: 3},
		{Code: "s7", Title: "菱形", Difficulty: 3}, {Code: "s8", Title: "五角星", Difficulty: 2},
	}
	return []moduleSpec{
		{Code: "add10", Name: "20以内加法", Kps: adds},
		{Code: "sub10", Name: "20以内减法", Kps: subs},
		{Code: "shape", Name: "认识图形", Kps: shapes},
	}
}

func difficultyBySum(sum int) int {
	switch {
	case sum <= 5:
		return 1
	case sum <= 10:
		return 2
	case sum <= 15:
		return 2
	default:
		return 3
	}
}

func literacyModules() []moduleSpec {
	chars := []string{
		"一", "二", "三", "四", "五", "六", "七", "八", "九", "十",
		"人", "口", "手", "足", "目", "耳", "日", "月", "水", "火",
		"山", "石", "田", "土", "木", "禾", "米", "竹", "花", "草",
		"大", "小", "多", "少", "上", "下", "左", "右", "前", "后",
		"天", "地", "风", "云", "雨", "雪", "春", "夏", "秋", "冬",
		"爸", "妈", "爷", "奶", "哥", "姐", "弟", "妹", "家", "我",
		"你", "他", "她", "们", "的", "了", "不", "在", "有", "是",
		"来", "去", "看", "听", "说", "写", "读", "画", "唱", "跳",
		"红", "黄", "蓝", "绿", "白", "黑", "猫", "狗", "鸟", "鱼",
		"车", "船", "门", "窗", "书", "笔", "纸", "课", "学", "玩",
		"牛", "羊", "马", "鸡", "鸭", "虫", "蜂", "蝶", "蛙", "龟",
		"刀", "尺", "伞", "灯", "床", "桌", "椅", "碗", "勺", "杯",
		"金", "银", "铜", "铁", "线", "绳", "包", "盒", "瓶", "罐",
		// 大班补充：身体感受 / 穿戴 / 食物 / 园所时间 / 动作自然
		"头", "脸", "牙", "鼻", "心", "笑", "哭", "爱", "好", "乖",
		"衣", "帽", "鞋", "袜", "巾", "裤", "裙", "被", "枕", "袋",
		"饭", "菜", "肉", "蛋", "茶", "果", "苹", "桃", "瓜", "豆",
		"早", "晚", "今", "明", "年", "友", "园", "班", "师", "生",
		"走", "跑", "飞", "坐", "站", "星", "光", "电", "气", "开",
		// 大班再补：身体部位 / 自然景物 / 日常用具 / 方位 / 冷暖进出
		"眼", "嘴", "舌", "发", "脖", "肩", "臂", "指", "肚", "背",
		"树", "叶", "林", "河", "湖", "海", "沙", "路", "桥", "岛",
		"刷", "梳", "镜", "皂", "盆", "桶", "扫", "箱", "柜", "锁",
		"里", "外", "中", "旁", "边", "东", "西", "南", "北", "方",
		"冷", "热", "暖", "凉", "晴", "阴", "出", "回", "进", "到",
		// 大班续补：动物 / 食物 / 字词图画 / 时间先后 / 礼貌 / 生活动作 / 自然现象
		"兔", "鼠", "虎", "龙", "蛇", "猪", "猴", "熊", "狮", "象",
		"面", "汤", "糖", "饼", "饺", "油", "盐", "醋", "酱", "粥",
		"本", "页", "字", "词", "句", "文", "诗", "歌", "音", "图",
		"昨", "每", "次", "刚", "才", "正", "快", "慢", "先", "又",
		"请", "谢", "对", "起", "再", "见", "问", "答", "叫", "名",
		"吃", "喝", "睡", "醒", "洗", "穿", "脱", "拿", "给", "放",
		"雷", "雾", "冰", "虹", "霞", "露", "烟", "灰", "尘", "泥",
	}
	var mods []moduleSpec
	for i, group := range chunk(chars, 10) {
		diff := 1 + i/4
		if diff > 3 {
			diff = 3
		}
		mods = append(mods, moduleSpec{
			Code: fmt.Sprintf("g%d", i+1),
			Name: fmt.Sprintf("第%d组", i+1),
			Kps:  genKps(fmt.Sprintf("l%d", i+1), group, diff),
		})
	}
	return mods
}

func pinyinModules() []moduleSpec {
	shengmu := []string{
		"b", "p", "m", "f", "d", "t", "n", "l", "g", "k", "h",
		"j", "q", "x", "zh", "ch", "sh", "r", "z", "c", "s", "y", "w",
	}
	yunmu := []string{
		"a", "o", "e", "i", "u", "ü", "ai", "ei", "ui", "ao", "ou",
		"iu", "ie", "üe", "er", "an", "en", "in", "un", "ün", "ang", "eng",
	}
	return []moduleSpec{
		{Code: "shengmu", Name: "声母", Kps: genKps("sm", shengmu, 1)},
		{Code: "yunmu", Name: "韵母", Kps: genKps("ym", yunmu, 2)},
	}
}

func englishModules() []moduleSpec {
	topics := []struct {
		code  string
		name  string
		words []string
	}{
		{"animals", "Animals", []string{"cat", "dog", "bird", "fish", "rabbit", "tiger", "lion", "bear", "monkey", "panda"}},
		{"colors", "Colors", []string{"red", "blue", "green", "yellow", "pink", "black", "white", "orange", "purple", "brown"}},
		{"numbers", "Numbers", []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}},
		{"food", "Food", []string{"apple", "banana", "bread", "milk", "egg", "rice", "cake", "juice", "soup", "candy"}},
		{"family", "Family", []string{"mom", "dad", "baby", "grandpa", "grandma", "brother", "sister", "uncle", "aunt", "cousin"}},
		{"body", "Body", []string{"head", "eye", "ear", "nose", "mouth", "hand", "foot", "arm", "leg", "hair"}},
		{"fruits", "Fruits", []string{"grape", "peach", "pear", "orange", "lemon", "melon", "cherry", "mango", "kiwi", "berry"}},
		{"weather", "Weather", []string{"sunny", "rainy", "cloudy", "windy", "snowy", "hot", "cold", "warm", "cool", "storm"}},
		{"toys", "Toys", []string{"ball", "doll", "car", "block", "kite", "train", "puzzle", "robot", "balloon", "slide"}},
		{"school", "School", []string{"book", "pen", "pencil", "bag", "desk", "chair", "teacher", "student", "class", "school"}},
		{"clothes", "Clothes", []string{"shirt", "pants", "dress", "hat", "shoe", "sock", "coat", "scarf", "glove", "skirt"}},
		{"actions", "Actions", []string{"run", "jump", "walk", "sit", "stand", "eat", "drink", "sleep", "read", "write"}},
		{"places", "Places", []string{"home", "park", "zoo", "shop", "farm", "beach", "library", "museum", "hospital", "cinema"}},
		{"transport", "Transport", []string{"bus", "bike", "plane", "boat", "train", "taxi", "truck", "subway", "helicopter", "ship"}},
		{"shapes", "Shapes", []string{"circle", "square", "triangle", "star", "heart", "oval", "rectangle", "diamond", "cross", "arrow"}},
		{"time", "Time", []string{"morning", "noon", "afternoon", "evening", "night", "today", "yesterday", "tomorrow", "week", "year"}},
		{"feelings", "Feelings", []string{"happy", "sad", "angry", "tired", "hungry", "thirsty", "scared", "brave", "kind", "funny"}},
		{"nature", "Nature", []string{"tree", "flower", "grass", "river", "mountain", "sun", "moon", "star", "cloud", "rain"}},
		{"jobs", "Jobs", []string{"doctor", "nurse", "chef", "pilot", "driver", "farmer", "singer", "dancer", "police", "firefighter"}},
		{"greetings", "Greetings", []string{"hello", "hi", "bye", "thanks", "please", "sorry", "yes", "no", "ok", "welcome"}},
	}
	var mods []moduleSpec
	for i, t := range topics {
		diff := 1 + i/7
		if diff > 3 {
			diff = 3
		}
		mods = append(mods, moduleSpec{Code: t.code, Name: t.name, Kps: genKps(fmt.Sprintf("e%d", i), t.words, diff)})
	}
	return mods
}

func gameModules() []moduleSpec {
	levels := make([]string, 8)
	for i := 0; i < 8; i++ {
		levels[i] = fmt.Sprintf("第%d关", i+1)
	}
	return []moduleSpec{{Code: "levels", Name: "闯关", Kps: genKps("lv", levels, 2)}}
}
