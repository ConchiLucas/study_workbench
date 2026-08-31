package sense

import "fmt"

// PromptScience builds a science-concept sense-image prompt (same sticker style as literacy).
func PromptScience(title string) string {
	subject := scienceSubject(title)
	p := fmt.Sprintf(
		"Create a clean, child-friendly cartoon illustration of %s. %s",
		subject,
		styleSuffix,
	)
	if ContainsHan(p) {
		return fmt.Sprintf(
			"Create a clean, child-friendly cartoon illustration of a simple child-friendly cartoon science scene for young children, with no writing on it. %s",
			styleSuffix,
		)
	}
	return p
}

func scienceSubject(title string) string {
	if s, ok := scienceSubjects[title]; ok {
		return s
	}
	return fmt.Sprintf("a single simple child-friendly cartoon scene that clearly shows the science concept %q for young children, with no writing on it", title)
}

var scienceSubjects = map[string]string{
	// animal
	"哺乳动物": "a mother cat nursing her kitten, showing mammals drink milk",
	"鸟类":     "a colorful cartoon bird with feathers and wings spread",
	"鱼类":     "a cartoon fish swimming underwater with visible gills",
	"昆虫":     "a cute cartoon bee with six legs and antennae",
	"爬行动物": "a cartoon turtle with a shell and scaly skin",
	"两栖动物": "a cartoon frog with a tadpole in water nearby",
	"食草动物": "a friendly cartoon rabbit eating green leaves and grass",
	"食肉动物": "a friendly cartoon lion with sharp teeth",
	"冬眠":     "a cartoon bear sleeping inside a cozy cave in winter snow",
	"迁徙":     "a flock of cartoon swallows flying south over mountains",

	// plant
	"种子":   "a seed sprouting from brown soil with a tiny green shoot",
	"根":     "plant roots underground absorbing water, with green leaves above",
	"茎":     "a green plant stem supporting leaves and a flower",
	"叶":     "a large green leaf catching sunlight",
	"花":     "a bright pink flower with petals and a green stem",
	"果实":   "a red apple and a yellow banana as plant fruits",
	"光合作用": "green leaves under a bright sun with arrows suggesting energy",
	"树木":   "a tall tree with a thick brown trunk and green canopy",
	"蔬菜":   "carrots, broccoli, and a cucumber as vegetables",
	"水果":   "grapes and strawberries as sweet fruits",

	// weather
	"晴天": "a bright yellow sun in a clear blue sky",
	"雨天": "raindrops falling from a gray cloud",
	"阴天": "a gray overcast sky blocking the sun",
	"雪天": "white snowflakes falling from a cloud",
	"风":   "curved wind lines blowing leaves off a tree",
	"雷电": "a storm cloud with a yellow lightning bolt",
	"彩虹": "a colorful rainbow arc after rain",
	"温度": "a cartoon thermometer showing hot and cold",
	"湿度": "water droplets in the air near a cloud",
	"四季": "four small panels: spring flower, summer sun, autumn leaf, winter snow",

	// space
	"太阳": "a bright yellow sun with rays",
	"月亮": "a yellow crescent moon in a night sky",
	"星星": "twinkling stars scattered in a dark sky",
	"地球": "a blue and green cartoon Earth globe",
	"行星": "planets orbiting the sun in space",
	"银河": "a spiral galaxy with many stars",
	"昼夜": "half sun and half moon showing day and night",
	"潮汐": "ocean waves rising and falling on a beach",
	"日食": "the moon passing in front of the sun during daytime",
	"月食": "Earth's shadow falling on the moon at night",

	// body
	"骨骼": "a friendly cartoon skeleton showing bones",
	"肌肉": "an arm flexing to show strong muscles",
	"心脏": "a red cartoon heart beating",
	"肺":   "a pair of pink cartoon lungs inflating",
	"大脑": "a pink cartoon brain with a lightbulb idea",
	"消化": "food traveling through a simple stomach and intestines",
	"五官": "a friendly face showing eyes, ears, nose, and mouth",
	"牙齿": "a row of white cartoon teeth in a smile",
	"血液": "red blood cells flowing through a vein",
	"皮肤": "a hand touching skin to feel temperature",

	// traffic
	"红绿灯": "a traffic light with red, yellow, and green lights",
	"斑马线": "a crosswalk with white stripes on a road",
	"安全带": "a seat belt buckled in a car seat",
	"头盔":   "a bicycle safety helmet",
	"人行道": "a sidewalk path separate from a road",
	"公交":   "a yellow city bus at a bus stop",
	"地铁":   "a subway train in an underground tunnel",
	"自行车": "a child riding a bicycle with a helmet",
	"汽车":   "a simple family car on a road",
	"火车":   "a long passenger train on railroad tracks",

	// eco
	"垃圾分类": "colorful recycling bins sorted by type",
	"回收":     "old bottles and paper going into a recycling bin",
	"节约用水": "a child turning off a faucet while brushing teeth",
	"节约用电": "a hand switching off a light when leaving a room",
	"植树":     "a child planting a small tree sapling",
	"空气污染": "gray smoke clouds over a city skyline",
	"水污染":   "dirty water with trash floating in a river",
	"塑料":     "colorful plastic bottles and bags",
	"电池":     "AA batteries powering a toy",
	"绿色出行": "a child walking and cycling instead of driving",

	// safety
	"防火":     "a fire extinguisher and a smoke alarm",
	"防电":     "a wet hand staying away from an electrical outlet",
	"防溺水":   "a no-swimming sign at a deep river without adults",
	"防走失":   "a child waiting at a help desk looking for a parent",
	"急救电话": "a red emergency telephone with a large green call button",
	"地震":     "a child crouching under a sturdy table during shaking",
	"台风":     "strong wind and rain with a house window closed",
	"烫伤":     "running cool water over a minor burn on a hand",
	"割伤":     "a bandage being applied to a small finger cut",
	"陌生人":   "a child stepping back from an unknown person offering candy",

	// material
	"木头": "a wooden table and chair",
	"石头": "a rough gray rock or boulder",
	"金属": "shiny metal keys and a spoon",
	"玻璃": "a clear glass window pane",
	"纸":   "sheets of white paper for drawing",
	"布":   "folded colorful fabric for clothes",
	"橡胶": "a rubber band and a tire showing stretchiness",
	"磁铁": "a horseshoe magnet attracting paper clips",
	"沙子": "fine yellow sand in a beach bucket",

	// life
	"刷牙":     "a child brushing teeth with toothpaste",
	"洗手":     "hands being washed with soap under a faucet",
	"早睡早起": "a child sleeping at night and waking up with the sun",
	"均衡饮食": "a balanced plate with vegetables, rice, and meat",
	"运动":     "children running and playing ball outdoors",
	"喝水":     "a glass of plain water after exercise",
	"防晒":     "a child wearing a sun hat and sunscreen outdoors",
	"穿衣":     "a child putting on a warm jacket in cold weather",
	"整理玩具": "toys being put back into a storage box neatly",
}
