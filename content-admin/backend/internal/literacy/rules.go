package literacy

// NeedsSenseImage 按确定性词表判断该字是否需要义图。
func NeedsSenseImage(char string) bool {
	if _, no := noSenseImageChars[char]; no {
		return false
	}
	return true
}

// 虚词 / 指代 / 判断等：没有稳定可画的实物含义。
var noSenseImageChars = map[string]struct{}{
	"的": {}, "了": {}, "不": {}, "在": {}, "有": {}, "是": {},
	"你": {}, "他": {}, "她": {}, "们": {},
	"这": {}, "那": {}, "吗": {}, "呢": {}, "吧": {}, "着": {}, "过": {},
	"和": {}, "与": {}, "也": {}, "很": {}, "就": {}, "都": {},
	"把": {}, "让": {}, "从": {}, "向": {},
	"比": {}, "为": {}, "以": {}, "而": {}, "且": {}, "或": {},
}
