# 《The Zoo Treasure Hunt》短对话版设计

## 目标

重写现有故事，使其成为适合小学三年级学生朗读和表演的儿童英语短剧。故事必须覆盖小学英语三年级上册的 63 个目标词，同时避免长句和词汇堆砌。

## 输出文件

- 覆盖更新：`小学英语/3年级上册/The Zoo Treasure Hunt.md`
- 文件格式：Markdown。
- 保留英文标题、中文标题和文末 63 词核对表。

## 表现形式

- 正文采用纯角色对话，不使用长篇叙述段落。
- 使用简短场景标题交代地点和故事阶段，场景标题不参与英文台词词数统计。
- 每句英文台词后紧跟对应的中文翻译。
- 角色包括 Mia、Ben、Mum、Panda、Monkey、Tiger 等；角色名不参与台词词数统计。

## 台词约束

- 每句英文台词最多 5 个英文单词。
- 优先使用约 3 个英文单词的台词。
- Markdown 加粗符号和标点不参与单词计数。
- 缩写按一个英文单词计数，例如 `Let's` 计为一个词。
- 每一行只放一句角色台词，避免用标点连接多个句子。

## 故事结构

1. 出发：Mia 在 Mum 和 Ben 的提醒下收拾书包及文具。
2. 动物园入口：Mia 到达动物园并接受 Panda 的寻宝邀请。
3. 动物朋友：不同动物通过对话加入寻宝。
4. 颜色线索：角色寻找红、蓝、绿、黄、黑、白、棕和橙色物品。
5. 数字线索：角色从一数到十，打开下一道门。
6. 身体线索：角色通过指出身体部位完成动作挑战。
7. 野餐与宝藏：大家分享食物和饮料，最终发现“友谊是宝藏”。

## 词汇要求

- 以下 63 个目标词必须全部在英文对话正文中出现并使用粗体标记：

```text
arm, bag, bear, bird, black, blue, body, book, bread, brother, brown, cake, cat, crayon, dog, duck, ear, egg, eight, elephant, eraser, eye, face, fish, five, foot, four, funny, green, hand, head, juice, leg, milk, monkey, mouth, mum, nine, no, nose, OK, one, orange, panda, pen, pencil, pig, plate, red, rice, ruler, school, seven, six, ten, three, tiger, two, water, white, yellow, your, zoo
```

- 可以使用少量简单的非目标词连接情节。
- 目标词必须放在有意义的对话语境中，不能单独罗列充当正文。

## 自动验收

- 正文包含 7 个场景。
- 每句英文台词都有相邻的中文翻译。
- 英文正文覆盖全部 63 个目标词。
- 任意英文台词均不超过 5 个英文单词。
- 统计所有英文台词的平均词数，并以接近 3 个词为优化目标。
- 文末核对表包含编号 1—63，顺序和目标词表一致。
