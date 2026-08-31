package plan

import "testing"

// gen 造 n 个候选，均匀分配给给定的学科。
func gen(bucket string, subjects []string, n int, startID int64) []Candidate {
	out := make([]Candidate, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, Candidate{
			KpID:        startID + int64(i),
			SubjectCode: subjects[i%len(subjects)],
			Bucket:      bucket,
		})
	}
	return out
}

func countByBucket(cs []Candidate) map[string]int {
	out := map[string]int{}
	for _, c := range cs {
		out[c.Bucket]++
	}
	return out
}

func countBySubject(cs []Candidate) map[string]int {
	out := map[string]int{}
	for _, c := range cs {
		out[c.SubjectCode]++
	}
	return out
}

func TestQuotaIsRespectedWhenAllBucketsAreFull(t *testing.T) {
	subjects := []string{"math", "literacy", "pinyin", "english"}
	var all []Candidate
	all = append(all, gen(BucketReview, subjects, 20, 1000)...)
	all = append(all, gen(BucketShaky, subjects, 20, 2000)...)
	all = append(all, gen(BucketLearning, subjects, 20, 3000)...)
	all = append(all, gen(BucketNew, subjects, 20, 4000)...)

	got := Compose(all, DefaultQuota(), DefaultRules(), 0)
	if len(got) != 10 {
		t.Fatalf("题量 = %d，期望 10", len(got))
	}

	buckets := countByBucket(got)
	want := map[string]int{BucketReview: 4, BucketShaky: 2, BucketLearning: 2, BucketNew: 2}
	for b, n := range want {
		if buckets[b] != n {
			t.Errorf("%s 桶 = %d，期望 %d", b, buckets[b], n)
		}
	}
}

func TestDeficitIsBackfilledToKeepPlanFull(t *testing.T) {
	// 刚开始用的场景：一条到期复习都没有，也没有易错点。
	subjects := []string{"math", "literacy"}
	var all []Candidate
	all = append(all, gen(BucketLearning, subjects, 3, 3000)...)
	all = append(all, gen(BucketNew, subjects, 30, 4000)...)

	got := Compose(all, DefaultQuota(), DefaultRules(), 0)
	if len(got) != 10 {
		t.Fatalf("题量 = %d，缺口应该由其他桶顶上凑满 10", len(got))
	}

	buckets := countByBucket(got)
	if buckets[BucketReview] != 0 || buckets[BucketShaky] != 0 {
		t.Errorf("没有候选的桶不该出现: %+v", buckets)
	}
	// 进行中只有 3 个候选，全部用上；剩下 7 题由新知补齐。
	if buckets[BucketLearning] != 3 {
		t.Errorf("进行中 = %d，期望 3（候选全用上）", buckets[BucketLearning])
	}
	if buckets[BucketNew] != 7 {
		t.Errorf("新知 = %d，期望 7", buckets[BucketNew])
	}
}

func TestPlanSpansAtMostThreeSubjects(t *testing.T) {
	// 英语候选远多于其他学科，模拟真实的知识点分布。
	var all []Candidate
	all = append(all, gen(BucketReview, []string{"english"}, 40, 1000)...)
	all = append(all, gen(BucketShaky, []string{"english", "literacy"}, 20, 2000)...)
	all = append(all, gen(BucketLearning, []string{"math", "pinyin"}, 20, 3000)...)
	all = append(all, gen(BucketNew, []string{"english", "math", "pinyin", "literacy"}, 40, 4000)...)

	got := Compose(all, DefaultQuota(), DefaultRules(), 0)
	if len(got) != 10 {
		t.Fatalf("题量 = %d", len(got))
	}

	bySubject := countBySubject(got)
	if len(bySubject) > 3 {
		t.Errorf("涉及 %d 个学科，超过上限 3: %+v", len(bySubject), bySubject)
	}
	for subj, n := range bySubject {
		if n > 5 {
			t.Errorf("%s 有 %d 题，超过单科上限 5", subj, n)
		}
	}
}

func TestSingleSubjectPlanRelaxesPerSubjectCap(t *testing.T) {
	// 只有一个学科有候选时，5 题的单科上限必须放开，否则计划凑不满。
	all := gen(BucketNew, []string{"english"}, 30, 4000)

	got := Compose(all, DefaultQuota(), DefaultRules(), 0)
	if len(got) != 10 {
		t.Fatalf("题量 = %d，单科上限应该在凑不满时放开", len(got))
	}
	if n := countBySubject(got)["english"]; n != 10 {
		t.Errorf("english = %d，期望 10", n)
	}
}

func TestSubjectsAreInterleavedNotClustered(t *testing.T) {
	// 同一个桶内应该在学科之间轮流取，而不是先把一个学科取完。
	all := gen(BucketNew, []string{"math", "literacy", "pinyin"}, 30, 4000)

	got := Compose(all, DefaultQuota(), DefaultRules(), 0)
	bySubject := countBySubject(got)
	if len(bySubject) != 3 {
		t.Fatalf("应该摊到 3 个学科，实际 %+v", bySubject)
	}
	for subj, n := range bySubject {
		if n < 3 || n > 4 {
			t.Errorf("%s 有 %d 题，3 个学科 10 题应该接近均分", subj, n)
		}
	}
}

func TestRotationChangesSubjectMix(t *testing.T) {
	subjects := []string{"math", "literacy", "pinyin", "english"}
	var all []Candidate
	all = append(all, gen(BucketReview, subjects, 40, 1000)...)
	all = append(all, gen(BucketShaky, subjects, 40, 2000)...)
	all = append(all, gen(BucketLearning, subjects, 40, 3000)...)
	all = append(all, gen(BucketNew, subjects, 40, 4000)...)

	// 4 个学科、上限 3 个，轮换起点应该换掉被挤出去的那个学科。
	seen := map[string]bool{}
	for day := 0; day < 4; day++ {
		got := Compose(all, DefaultQuota(), DefaultRules(), day)
		if len(got) != 10 {
			t.Fatalf("第 %d 天题量 = %d", day, len(got))
		}
		mix := ""
		for _, s := range subjects {
			if countBySubject(got)[s] > 0 {
				mix += s + ","
			}
		}
		seen[mix] = true
	}
	if len(seen) < 2 {
		t.Errorf("连续 4 天的学科组合完全相同，rotation 没起作用: %v", seen)
	}
}

func TestNoDuplicateKnowledgePoints(t *testing.T) {
	// 同一个知识点同时出现在多个桶的候选里（数据异常），也不该在计划里出现两次。
	dup := []Candidate{
		{KpID: 1, SubjectCode: "math", Bucket: BucketReview},
		{KpID: 1, SubjectCode: "math", Bucket: BucketShaky},
		{KpID: 1, SubjectCode: "math", Bucket: BucketNew},
	}
	all := append(dup, gen(BucketNew, []string{"math", "literacy"}, 20, 100)...)

	got := Compose(all, DefaultQuota(), DefaultRules(), 0)
	seen := map[int64]bool{}
	for _, c := range got {
		if seen[c.KpID] {
			t.Errorf("知识点 %d 重复出现", c.KpID)
		}
		seen[c.KpID] = true
	}
}

func TestNotEnoughCandidatesYieldsShortPlan(t *testing.T) {
	// 候选就是不够时返回能给的，不编造——上层据此决定是否提示家长。
	all := gen(BucketNew, []string{"math"}, 4, 1)
	got := Compose(all, DefaultQuota(), DefaultRules(), 0)
	if len(got) != 4 {
		t.Errorf("题量 = %d，期望 4", len(got))
	}
}

func TestEmptyCandidatesYieldEmptyPlan(t *testing.T) {
	if got := Compose(nil, DefaultQuota(), DefaultRules(), 0); len(got) != 0 {
		t.Errorf("没有候选却生成了 %d 题", len(got))
	}
}

func TestDefaultQuotaTotalsTen(t *testing.T) {
	if n := DefaultQuota().Total(); n != 10 {
		t.Errorf("默认题量 = %d，期望 10", n)
	}
}
