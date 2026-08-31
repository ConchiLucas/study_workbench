import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import test from "node:test";

const componentUrl = new URL("../src/components/ClozeResultTable.tsx", import.meta.url);
const appUrl = new URL("../src/App.tsx", import.meta.url);

test("sentence table provides the approved main and expanded columns", () => {
  assert.equal(existsSync(componentUrl), true, "ClozeResultTable component must exist");
  const source = readFileSync(componentUrl, "utf8");

  assert.match(source, /aria-expanded=\{expanded\}/);
  assert.match(source, /来源单词/);
  assert.match(source, /答错时间/);
  assert.match(source, /入口 \/ 模式/);
  assert.match(source, /词库 \/ 难度/);
  assert.match(source, /词难度/);
  assert.doesNotMatch(source, /你的答案/);
  assert.doesNotMatch(source, /word\.selectedAnswer/);
  assert.match(source, /正确答案/);
  assert.match(source, /来源追溯/);
  assert.match(source, /item\.clozeSentence/);
  assert.match(source, /item\.explanationZh/);
  assert.doesNotMatch(source, /\.join\(", "\)/);
});

test("sentence table supports multi-row expansion and clears it for a new result context", () => {
  assert.equal(existsSync(componentUrl), true, "ClozeResultTable component must exist");
  const source = readFileSync(componentUrl, "utf8");

  assert.match(source, /useState<Set<number>>/);
  assert.match(
    source,
    /useEffect\(\(\) => \{[\s\S]*setExpandedIds\(new Set\(\)\);[\s\S]*\}, \[resetKey\]\);/,
  );
  assert.match(source, /onClick=\{\(\) => toggleExpanded\(item\.id\)\}/);
});

test("App renders the expandable component and no longer owns the old detail modal", () => {
  const source = readFileSync(appUrl, "utf8");
  const clozePageSource = source.slice(
    source.indexOf("function ClozeResultsPage"),
    source.indexOf("function formatFullTime"),
  );

  assert.match(source, /import ClozeResultTable from "\.\/components\/ClozeResultTable"/);
  assert.match(clozePageSource, /<ClozeResultTable/);
  assert.match(clozePageSource, /resetKey=\{resultResetKey\}/);
  assert.match(
    clozePageSource,
    /selectedUser\?\.userId \?\? "all"[\s\S]*resultKeyword\.trim\(\)[\s\S]*resultPage[\s\S]*resultPageSize/,
  );
  assert.doesNotMatch(clozePageSource, /const \[detailItem, setDetailItem\]/);
  assert.doesNotMatch(clozePageSource, /title="生成结果详情"/);
});
