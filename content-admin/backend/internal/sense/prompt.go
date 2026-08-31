package sense

import (
	"fmt"
	"unicode"
)

const styleSuffix = `Centered on a pure white background. Flat vector style, thick black outlines, bright solid colors, smooth rounded shapes, child-friendly sticker look. Show only the real-world object or scene meaning — never write any characters, letters, digits, calligraphy, labels, or logos. Do not shape the object like a Chinese character glyph. No shadows, no extra objects, no scenery background.`

const Negative = `text, letters, digits, numbers, Chinese characters, Japanese characters, calligraphy, glyph-shaped object, watermark, logo, label, caption, photorealistic, 3d render, busy background, multiple unrelated subjects, shadows on background`

// SubjectEN returns an English subject phrase for the literacy character.
// It must never include Chinese characters — models tend to draw them onto the image.
func SubjectEN(char string) string {
	if s, ok := subjects[char]; ok {
		return s
	}
	return "a single simple child-friendly cartoon object that clearly shows the everyday meaning of this vocabulary concept, with no writing on it"
}

func Prompt(char string) string {
	p := fmt.Sprintf(
		"Create a clean, child-friendly cartoon illustration of %s. %s",
		SubjectEN(char),
		styleSuffix,
	)
	if ContainsHan(p) {
		// Defense: never send Han script to image models (they draw the glyph).
		return fmt.Sprintf(
			"Create a clean, child-friendly cartoon illustration of a single everyday object matching this concept id %d. %s",
			[]rune(char)[0],
			styleSuffix,
		)
	}
	return p
}

// ContainsHan reports whether s includes any Han script rune (for tests / guards).
func ContainsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// Concrete subjects: English only. Prefer countable / tangible objects over glyph metaphors.
var subjects = map[string]string{
	// g1 numbers
	"一": "exactly one large red apple with a green leaf, alone in the center",
	"二": "exactly two identical large red apples with green leaves, side by side in one neat row",
	"三": "exactly three identical large red apples with green leaves, in one neat horizontal row",
	"四": "exactly four identical large red apples with green leaves, in one neat horizontal row",
	"五": "exactly five identical large red apples with green leaves, in one neat horizontal row",
	"六": "exactly six identical yellow five-pointed stars, arranged in two neat rows of three",
	"七": "exactly seven identical yellow five-pointed stars, arranged in two neat rows (top row of four stars, bottom row of three stars), evenly spaced and easy to count",
	"八": "exactly eight identical yellow five-pointed stars, arranged in two neat rows of four",
	"九": "exactly nine identical blue round balloons with short strings, arranged in a neat 3-by-3 grid",
	"十": "exactly ten identical green leaves, arranged in two neat rows of five",

	// g2 body / nature basics
	"人": "a friendly standing cartoon child with arms slightly open",
	"口": "an open smiling mouth only (lips and teeth), not a full face",
	"手": "a single open hand with palm facing forward",
	"足": "a single bare human foot from the side",
	"目": "one large friendly cartoon eye",
	"耳": "a single human ear",
	"日": "a bright yellow sun with simple short rays",
	"月": "a yellow crescent moon",
	"水": "a single blue water droplet",
	"火": "a bright orange and yellow flame",

	// g3 nature
	"山": "green mountains with a small white snow tip",
	"石": "a gray rough rock / boulder with uneven rocky surface and small cracks, clearly a stone not a smooth circle",
	"田": "a simple green farm field divided into four squares",
	"土": "a small mound of brown soil",
	"木": "a simple green tree with a brown trunk",
	"禾": "a stalk of ripe yellow grain",
	"米": "a ceramic bowl filled with cooked white rice, grains clearly visible on top",
	"竹": "a green bamboo stalk with leaves",
	"花": "a pink five-petal flower with a short green stem",
	"草": "a small clump of green grass",

	// g4 size / position
	"大": "a large friendly elephant standing, looking big",
	"小": "a tiny cute mouse, looking small",
	"多": "a big pile of many colorful balls, looking plentiful",
	"少": "only two small colorful balls, looking few",
	"上": "a simple upward-pointing arrow sticker",
	"下": "a simple downward-pointing arrow sticker",
	"左": "a simple left-pointing arrow sticker",
	"右": "a simple right-pointing arrow sticker",
	"前": "a child walking forward away from the viewer",
	"后": "a child walking away showing their back",

	// g5 weather / seasons
	"天": "a bright blue sky with a few white fluffy clouds",
	"地": "brown ground with a patch of green grass",
	"风": "curved blue wind lines blowing past a small leaf",
	"云": "one fluffy white cloud",
	"雨": "blue raindrops falling from a small gray cloud",
	"雪": "white snowflakes falling",
	"春": "a pink cherry blossom branch",
	"夏": "a bright sun over green leaves",
	"秋": "an orange maple leaf",
	"冬": "a simple snowman",

	// g6 family
	"爸": "a friendly adult man father figure smiling",
	"妈": "a friendly adult woman mother figure smiling",
	"爷": "a friendly elderly grandfather smiling",
	"奶": "a friendly elderly grandmother smiling",
	"哥": "a friendly older brother boy smiling",
	"姐": "a friendly older sister girl smiling",
	"弟": "a friendly younger brother boy smiling",
	"妹": "a friendly younger sister girl smiling",
	"家": "a simple cute house with a door and windows",
	"我": "a friendly cartoon child pointing to their own chest with a thumb, meaning me / myself",

	// g8 actions
	"来": "a child walking toward the viewer waving",
	"去": "a child walking away to the side",
	"看": "a friendly cartoon child looking forward with wide open eyes, simple head and face only, clearly the action of looking / seeing",
	"听": "a child cupping an ear to listen",
	"说": "a speech bubble next to an open mouth (bubble empty, no letters)",
	"写": "a hand holding a pencil writing on blank paper (no letters on paper)",
	"读": "an open book with blank pages (no letters)",
	"画": "a paintbrush painting a simple flower on a blank canvas",
	"唱": "a child singing with musical notes floating (notes only, no letters)",
	"跳": "a child jumping happily in the air",

	// g9 colors / animals
	"红": "a solid bright red circle sticker",
	"黄": "a solid bright yellow circle sticker",
	"蓝": "a solid bright blue circle sticker",
	"绿": "a solid bright green circle sticker",
	"白": "a solid white circle with a thin black outline",
	"黑": "exactly one solid pure black circle sticker alone in the center on white, thick black outline optional, no other colors, no petals, no decorations",
	"猫": "a cute cartoon cat sitting",
	"狗": "a cute cartoon dog sitting",
	"鸟": "a cute cartoon bird perched",
	"鱼": "a cute cartoon fish",

	// g10 school / transport
	"车": "a simple cartoon car",
	"船": "a simple cartoon boat on a tiny wave",
	"门": "a closed wooden door with a round doorknob",
	"窗": "a house window with a brown wooden frame, four glass panes, and blue sky visible through the glass",
	"书": "a closed colorful book (cover blank, no title text)",
	"笔": "a single pencil",
	"纸": "a stack of white paper sheets with one corner slightly curled up, light gray edges so it is visible on white",
	"课": "a school desk with a blank notebook",
	"学": "a child reading a blank open book",
	"玩": "children toys: a ball and building blocks",

	// g11 animals
	"牛": "a friendly cartoon cow",
	"羊": "a friendly cartoon sheep",
	"马": "a friendly cartoon horse",
	"鸡": "a friendly cartoon chicken",
	"鸭": "a friendly cartoon duck",
	"虫": "a cute cartoon caterpillar crawling in a natural worm shape",
	"蜂": "a cute cartoon bee",
	"蝶": "a colorful butterfly",
	"蛙": "a cute cartoon frog",
	"龟": "a cute cartoon turtle",

	// g12 tools / furniture
	"刀": "a simple kitchen knife",
	"尺": "a yellow wooden ruler with only plain tick marks (no digits, no letters)",
	"伞": "an open colorful umbrella",
	"灯": "a simple table lamp glowing",
	"床": "a simple bed with a pillow",
	"桌": "a simple wooden table",
	"椅": "a simple chair",
	"碗": "a ceramic bowl",
	"勺": "a spoon",
	"杯": "a drinking cup",

	// g13 materials / containers
	"金": "a shiny gold bar / gold nugget (plain metal, no engravings)",
	"银": "a shiny silver coin (blank faces, no emblems or writing)",
	"铜": "a shiny copper pipe segment or copper nugget (plain metal, no writing)",
	"铁": "a plain gray iron nail",
	"线": "a spool of colorful thread",
	"绳": "a coiled rope",
	"包": "a simple school backpack",
	"盒": "a closed gift box with a ribbon (no text)",
	"瓶": "a glass bottle",
	"罐": "a round jar with a lid (blank label area, no text)",

	// g14 body / feelings
	"头": "a child's head in profile",
	"脸": "a friendly round smiling face",
	"牙": "a single white tooth",
	"鼻": "a simple cartoon nose",
	"心": "a red heart shape",
	"笑": "a big smiling face",
	"哭": "a crying face with tears",
	"爱": "two hands forming a heart shape",
	"好": "a green thumbs-up hand",
	"乖": "a well-behaved child sitting politely",

	// g15 clothes
	"衣": "a simple shirt",
	"帽": "a simple hat",
	"鞋": "a pair of shoes",
	"袜": "a pair of socks",
	"巾": "a folded towel",
	"裤": "a pair of pants",
	"裙": "a simple dress or skirt",
	"被": "a folded quilt / comforter for a bed (bedding blanket), clearly a physical quilt",
	"枕": "a fluffy pillow",
	"袋": "a simple tote bag (blank, no text)",

	// g16 food
	"饭": "a bowl of steamed rice",
	"菜": "a plate of green vegetables",
	"肉": "a piece of cooked meat",
	"蛋": "a fried egg",
	"茶": "a teacup with tea",
	"果": "assorted fruits: apple and banana",
	"苹": "a red apple",
	"桃": "a pink peach",
	"瓜": "a green watermelon slice",
	"豆": "a few green soybeans or peas",

	// g17 time / school people
	"早": "a sunrise over a horizon",
	"晚": "a night sky with a moon",
	"今": "a bright morning sunrise over a simple house rooftop, clearly meaning today / this morning",
	"明": "a bright morning sun",
	"年": "a birthday cake with candles",
	"友": "two children holding hands as friends",
	"园": "a colorful kindergarten playground with one slide and one swing on green grass",
	"班": "several children sitting in a classroom circle",
	"师": "a friendly teacher holding a book (blank cover)",
	"生": "a school student with a backpack",

	// g18 actions / energy
	"走": "a child walking",
	"跑": "a child running",
	"飞": "a single cartoon bird soaring with wide open wings against a light sky",
	"坐": "a child sitting on a chair",
	"站": "a child standing straight and still with feet together, arms at sides, clearly the action of standing",
	"星": "a bright yellow star",
	"光": "a glowing light bulb",
	"电": "a yellow lightning bolt",
	"气": "a puff of air / steam cloud",
	"开": "an open door",

	// g19 body parts
	"眼": "a pair of friendly eyes",
	"嘴": "closed lips forming a smile",
	"舌": "a cartoon tongue",
	"发": "a lock of hair or hairstyle",
	"脖": "a neck with a scarf",
	"肩": "shoulders of a child",
	"臂": "an arm flexed",
	"指": "a pointing finger",
	"肚": "a round belly",
	"背": "a child's back",

	// g20 nature places
	"树": "a leafy green tree",
	"叶": "a single green leaf",
	"林": "a small group of three trees",
	"河": "a blue river winding",
	"湖": "a calm blue lake",
	"海": "blue ocean waves",
	"沙": "a mound of yellow sand",
	"路": "a simple paved road",
	"桥": "a simple arched bridge",
	"岛": "a small island with one palm tree",

	// g21 household
	"刷": "a toothbrush",
	"梳": "a hair comb",
	"镜": "a round hand mirror (reflection blank)",
	"皂": "a bar of soap",
	"盆": "a washbasin",
	"桶": "a bucket",
	"扫": "a broom",
	"箱": "a storage box (blank, no text)",
	"柜": "a cabinet with doors",
	"锁": "a padlock",

	// g22 location / directions
	"里": "the inside of an open box looking inward",
	"外": "the outdoors: sun and tree outside a doorway",
	"中": "a target bullseye in the center",
	"旁": "a cat sitting beside a box",
	"边": "the edge of a table with a cup near the edge",
	"东": "a sunrise on the right side of the sky",
	"西": "a sunset on the left side of the sky",
	"南": "a warm sun high above",
	"北": "a snowflake suggesting north cold",
	"方": "a simple square shape",

	// g23 weather / motion
	"冷": "an ice cube",
	"热": "steam rising from a hot bowl",
	"暖": "a cozy scarf and mittens",
	"凉": "a cool breeze with a leaf",
	"晴": "a clear sunny sky",
	"阴": "a gray overcast cloud covering the sun",
	"出": "a chick coming out of an eggshell",
	"回": "a curved return arrow (no letters)",
	"进": "a person stepping in through an open door",
	"到": "a destination flag on a map pin (blank flag, no text)",

	// g24 animals
	"兔": "a cute cartoon rabbit",
	"鼠": "a cute cartoon mouse",
	"虎": "a friendly cartoon tiger",
	"龙": "a friendly cartoon Chinese dragon (creature only, no writing)",
	"蛇": "a cute cartoon snake",
	"猪": "a cute cartoon pig",
	"猴": "a cute cartoon monkey",
	"熊": "a cute cartoon bear",
	"狮": "a cute cartoon lion",
	"象": "a cute cartoon elephant",

	// g25 food staples
	"面": "a bowl of noodles",
	"汤": "a bowl of soup",
	"糖": "a few colorful hard candies (no wrappers with text)",
	"饼": "a round flat pancake or cookie",
	"饺": "a few dumplings on a plate",
	"油": "a bottle of cooking oil (blank label)",
	"盐": "a small salt shaker (blank, no text)",
	"醋": "a small vinegar bottle (blank label)",
	"酱": "a jar of sauce (blank label)",
	"粥": "a bowl of congee / porridge",

	// g26 literacy objects — never draw written characters on them
	"本": "a thick hardcover blank notebook standing upright, plain cover with no text",
	"页": "one blank paper page flipping open from a book, page completely empty",
	"字": "a child carefully writing one simple stroke with a pencil on blank paper (no letters or characters visible)",
	"词": "two picture flashcards side by side showing a cat and a dog (no letters on cards)",
	"句": "three small picture cards in a row telling a mini story (sun, flower, bee), completely blank of writing",
	"文": "a child writing on a stack of blank lined paper with a pencil, paper has empty lines only",
	"诗": "an open blank poetry book under a flowering peach branch with a crescent moon nearby (no writing in the book)",
	"歌": "musical notes floating near a microphone (notes only, no letters)",
	"音": "a speaker emitting sound waves",
	"图": "a simple picture frame with a landscape drawing inside (no captions)",

	// g27 time / manner
	"昨": "a calendar page flipped back (blank, no digits)",
	"每": "a repeating row of three identical apples",
	"次": "a simple counter tally of three blank sticks",
	"刚": "a stopwatch just stopped (blank face, no digits)",
	"才": "a small sprout just emerging from soil",
	"正": "a checkmark badge",
	"快": "a running rabbit suggesting speed",
	"慢": "a crawling snail suggesting slowness",
	"先": "a gold medal in first place (blank, no number)",
	"又": "two identical apples side by side meaning again",

	// g28 manners / speech
	"请": "two open hands offering politely",
	"谢": "a child bowing thank-you with hands together",
	"对": "a big green checkmark meaning correct / right",
	"起": "a child standing up from a chair",
	"再": "a circular redo arrow (no letters)",
	"见": "two children waving hello",
	"问": "a child raising a hand to ask a question",
	"答": "a child speaking with an empty speech bubble (no letters)",
	"叫": "a megaphone",
	"名": "a blank name tag sticker (completely blank)",

	// g29 daily actions
	"吃": "a child eating from a bowl with a spoon",
	"喝": "a child drinking from a cup",
	"睡": "a child sleeping in bed with closed eyes",
	"醒": "a child waking up stretching",
	"洗": "hands being washed under a faucet",
	"穿": "a child putting on a jacket",
	"脱": "a child taking off a shoe",
	"拿": "a hand picking up an apple",
	"给": "one hand giving an apple to another open hand",
	"放": "a hand placing a cup on a table",

	// g30 weather / nature bits
	"雷": "a cloud with a yellow lightning bolt",
	"雾": "soft gray fog swirls",
	"冰": "a clear ice cube",
	"虹": "a colorful rainbow arc",
	"霞": "pink and orange sunset glow clouds",
	"露": "a dew drop on a green leaf",
	"烟": "a thin puff of gray smoke",
	"灰": "a small pile of gray ash",
	"尘": "a dust cloud puff",
	"泥": "a brown mud puddle",
}
