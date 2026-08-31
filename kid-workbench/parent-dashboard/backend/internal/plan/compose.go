// Package plan 决定一份每日答题计划里放哪些知识点。
//
// 纯函数，不碰数据库：输入候选知识点（已按"该练的程度"排好序），
// 输出选中的若干条。这样配额和学科分布的规则可以直接用单测覆盖。
package plan

// 四个桶，按优先级从高到低。
const (
	BucketReview   = "review"   // 曾掌握、到复习点了，最该练
	BucketShaky    = "shaky"    // 反复出错的重点
	BucketLearning = "learning" // 练过但还不稳
	BucketNew      = "new"      // 没学过的，每天来一点保持新鲜感
)

// Quota 是各桶的目标题数。总和即一份计划的题量。
type Quota struct {
	Review   int
	Shaky    int
	Learning int
	New      int
}

// DefaultQuota 合计 10 题。这个数是从 5 岁孩子的注意力窗口倒推的：
// 专注上限约 8~12 分钟，每题含读题、思考、点选约 30~45 秒。
func DefaultQuota() Quota {
	return Quota{Review: 4, Shaky: 2, Learning: 2, New: 2}
}

func (q Quota) Total() int { return q.Review + q.Shaky + q.Learning + q.New }

// Rules 约束学科分布。
type Rules struct {
	MaxSubjects   int
	MaxPerSubject int
}

// DefaultRules 限制一份计划最多 3 个学科、单科最多 5 题。
// 没有这条约束，光按优先级排会让整份计划被英语占满——英语有 200 个知识点，
// 是识字的 1.5 倍、算术的 3 倍。
func DefaultRules() Rules {
	return Rules{MaxSubjects: 3, MaxPerSubject: 5}
}

type Candidate struct {
	KpID        int64
	SubjectCode string
	Bucket      string
}

// Compose 从候选里挑出一份计划。
//
// candidates 必须已按桶内优先级排好序（越该练的排越前），Compose 只负责配额与分布。
// rotation 用于轮换起始学科（传当天的天数即可），让每天的学科组合不一样。
func Compose(candidates []Candidate, q Quota, r Rules, rotation int) []Candidate {
	if r.MaxSubjects <= 0 {
		r.MaxSubjects = DefaultRules().MaxSubjects
	}
	if r.MaxPerSubject <= 0 {
		r.MaxPerSubject = DefaultRules().MaxPerSubject
	}

	byBucket := map[string][]Candidate{}
	var subjects []string
	seen := map[string]bool{}
	for _, c := range candidates {
		byBucket[c.Bucket] = append(byBucket[c.Bucket], c)
		if !seen[c.SubjectCode] {
			seen[c.SubjectCode] = true
			subjects = append(subjects, c.SubjectCode)
		}
	}
	if n := len(subjects); n > 1 && rotation > 0 {
		k := rotation % n
		subjects = append(append([]string{}, subjects[k:]...), subjects[:k]...)
	}

	st := &picker{
		limit:    q.Total(),
		rules:    r,
		subjects: subjects,
		perSubj:  map[string]int{},
		taken:    map[int64]bool{},
	}

	for _, b := range []struct {
		bucket string
		n      int
	}{
		{BucketReview, q.Review},
		{BucketShaky, q.Shaky},
		{BucketLearning, q.Learning},
		{BucketNew, q.New},
	} {
		st.fill(byBucket[b.bucket], b.n, false)
	}

	// 某个桶候选不够（刚开始用的时候一条 review_due 都没有），
	// 缺口按 巩固 → 进行中 → 新知 → 复习 依次顶上，保证每天都是满 10 题。
	for _, b := range []string{BucketShaky, BucketLearning, BucketNew, BucketReview} {
		st.fill(byBucket[b], st.remaining(), false)
	}

	// 还是不够，说明被学科上限卡住了。放开限制凑满——
	// 一天全是英语也比只有 6 题好，计划做不满会让孩子觉得任务没完成。
	for _, b := range []string{BucketReview, BucketShaky, BucketLearning, BucketNew} {
		st.fill(byBucket[b], st.remaining(), true)
	}

	return st.picked
}

type picker struct {
	limit    int
	rules    Rules
	subjects []string
	// cursor 是学科轮转的位置，跨 fill 调用保留。
	// 每个桶都从头轮一遍的话，排在前面的学科会被反复优先，
	// 三个学科十道题会摊成 4/4/2 而不是 4/3/3。
	cursor  int
	perSubj map[string]int
	taken   map[int64]bool
	picked  []Candidate
}

func (p *picker) remaining() int { return p.limit - len(p.picked) }

// fill 从 pool 里取最多 n 条，在学科之间轮流取，让题目在学科上摊开。
func (p *picker) fill(pool []Candidate, n int, relaxed bool) {
	if n <= 0 || len(p.subjects) == 0 {
		return
	}

	queues := map[string][]Candidate{}
	for _, c := range pool {
		if p.taken[c.KpID] {
			continue
		}
		queues[c.SubjectCode] = append(queues[c.SubjectCode], c)
	}

	added := 0
	// 连续空转一整圈说明剩下的学科要么没候选、要么已超限。
	misses := 0
	for added < n && misses < len(p.subjects) {
		subj := p.subjects[p.cursor]
		p.cursor = (p.cursor + 1) % len(p.subjects)

		queue := queues[subj]
		if len(queue) == 0 || (!relaxed && !p.canTake(subj)) {
			misses++
			continue
		}
		queues[subj] = queue[1:]
		p.take(queue[0])
		added++
		misses = 0
	}
}

func (p *picker) canTake(subject string) bool {
	if used, ok := p.perSubj[subject]; ok {
		return used < p.rules.MaxPerSubject
	}
	return len(p.perSubj) < p.rules.MaxSubjects
}

func (p *picker) take(c Candidate) {
	p.picked = append(p.picked, c)
	p.perSubj[c.SubjectCode]++
	p.taken[c.KpID] = true
}
