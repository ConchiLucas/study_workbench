# Remove CLI Add Buttons Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove every manual “新增 Codex / 新增 Gemini” entry from the local CLI tab while preserving existing CLI editing and deletion.

**Architecture:** This is a presentation-only React change. The execution configuration domain model, Go API, database rows, CLI bootstrap, active-target selection, and delete behavior remain unchanged.

**Tech Stack:** React, TypeScript, Ant Design, Node test runner, Vite

---

### Task 1: Remove manual CLI creation controls

**Files:**
- Modify: `word_select_dashboard/web-react/test/executionConfigPage.contract.test.ts`
- Modify: `word_select_dashboard/web-react/src/components/ExecutionConfigPage.tsx`

- [ ] **Step 1: Write the failing source-contract test**

Add this test to `executionConfigPage.contract.test.ts`:

```ts
test("CLI tab does not expose manual Codex or Gemini creation controls", () => {
  assert.doesNotMatch(componentSource, /新增 Codex/);
  assert.doesNotMatch(componentSource, /新增 Gemini/);
  assert.doesNotMatch(componentSource, /function addCLIProvider/);
  assert.match(componentSource, /<strong>暂无 CLI 配置<\/strong>/);
  assert.match(componentSource, /onClick=\{deleteCLIProvider\}>删除配置<\/Button>/);
});
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/executionConfigPage.contract.test.ts
```

Expected: FAIL because `ExecutionConfigPage.tsx` still contains both button labels and `addCLIProvider`.

- [ ] **Step 3: Remove the CLI-only creation UI and dead handler**

In `ExecutionConfigPage.tsx`:

- Remove `createDefaultCLIProvider` from the feature import.
- Delete the `addCLIProvider(driver: CLIDriver)` function.
- Pass `actions={null}` to the CLI `ProviderList`.
- Replace the empty CLI panel with:

```tsx
<div className="execution-inline-empty">
  <strong>暂无 CLI 配置</strong>
</div>
```

Do not change the API “新增 API” controls, `deleteCLIProvider`, CLI editing fields, or active-target behavior.

- [ ] **Step 4: Run focused and complete frontend verification**

Run:

```bash
cd word_select_dashboard/web-react
node --test --experimental-strip-types test/executionConfigPage.contract.test.ts
node --test --experimental-strip-types test/*.test.ts
npm run build
```

Expected: focused tests pass, all 65+ Node tests pass, and the Vite production build succeeds.

- [ ] **Step 5: Commit the implementation**

```bash
git add word_select_dashboard/web-react/test/executionConfigPage.contract.test.ts \
  word_select_dashboard/web-react/src/components/ExecutionConfigPage.tsx
git commit -m "fix: remove cli add buttons"
```

### Task 2: Rebuild and redeploy the management frontend

**Files:**
- Runtime-only image build; no additional tracked source files.

- [ ] **Step 1: Preserve the current frontend rollback image**

Run:

```bash
docker tag word-select-dashboard-web-react:1.0.0 \
  word-select-dashboard-web-react:pre-remove-cli-add-buttons
```

Expected: rollback tag points to the currently deployed image.

- [ ] **Step 2: Build the frontend image from local cached base layers**

Create ignored runtime file `.runtime/remove-cli-add-buttons/Dockerfile.web`:

```dockerfile
FROM word-select-dashboard-web-react:pre-remove-cli-add-buttons

COPY word_select_dashboard/web-react/dist/ /usr/share/nginx/html/
COPY word_select_dashboard/web-react/nginx.conf /etc/nginx/conf.d/default.conf
```

Run from the project root:

```bash
docker build --pull=false \
  -f .runtime/remove-cli-add-buttons/Dockerfile.web \
  -t word-select-dashboard-web-react:1.0.0 .
```

Expected: the image is built entirely from the local rollback image and current `dist/`.

- [ ] **Step 3: Recreate only the frontend service**

Run:

```bash
docker compose \
  --project-name word-select-dashboard-web \
  -f word_select_dashboard/web-react/docker-compose.yml \
  up -d --no-build
```

Expected: only `word-select-dashboard-web-react` is recreated; Go, Word Agent, CLI Runner, PostgreSQL, and TTS are untouched.

- [ ] **Step 4: Verify the deployed frontend**

Confirm:

```text
http://127.0.0.1:6016/ returns HTTP 200
word-select-dashboard-web-react container is running
the production bundle no longer contains “新增 Codex” or “新增 Gemini”
```

- [ ] **Step 5: Confirm unrelated working-tree changes remain untouched**

Run:

```bash
git status --short
```

Expected: only the user's pre-existing unrelated changes remain.
