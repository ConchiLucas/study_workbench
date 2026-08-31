import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const appSource = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");
const aiApiSource = readFileSync(new URL("../src/lib/aiConfigApi.ts", import.meta.url), "utf8");
const executionApiSource = readFileSync(new URL("../src/lib/executionConfigApi.ts", import.meta.url), "utf8");
const ttsApiSource = readFileSync(new URL("../src/lib/ttsConfigApi.ts", import.meta.url), "utf8");

test("provides login UI while keeping public read-only pages available", () => {
  assert.match(appSource, /function LoginPage/);
  assert.match(appSource, /管理员登录/);
  assert.match(appSource, /AUTH_REQUIRED_EVENT/);
  assert.match(appSource, /loginRequired/);
  assert.match(appSource, /onLogin/);
  assert.match(appSource, /onLogout/);
});

test("AI, execution, and TTS mutations share the authenticated request wrapper", () => {
  assert.match(aiApiSource, /requestJSON.*\.\/auth\.ts/);
  assert.match(executionApiSource, /requestJSON.*\.\/auth\.ts/);
  assert.match(ttsApiSource, /requestJSON.*\.\/auth\.ts/);
  assert.match(aiApiSource, /method:\s*"POST"/);
  assert.match(executionApiSource, /method:\s*"POST"/);
  assert.match(ttsApiSource, /method:\s*"POST"/);
});

test("auth challenge overlays the mounted tracker instead of replacing it", () => {
  const appComponent = appSource.slice(appSource.indexOf("export default function App"));
  assert.match(appComponent, /<TrackerApp[\s\S]*loginRequired\s*&&[\s\S]*<LoginPage/);
  assert.doesNotMatch(appComponent, /loginRequired\s*\?/);
  assert.match(appSource, /className="login-overlay"/);
});

test("root auth transitions clear private execution config cache even when its page is unmounted", () => {
  const appComponent = appSource.slice(appSource.indexOf("export default function App"));
  assert.match(appComponent, /const queryClient = useQueryClient\(\)/);
  assert.match(
    appComponent,
    /const requireLogin = \(\) => \{[\s\S]*removeQueries\(\{ queryKey: \["execution-config"\] \}\)/,
  );
  assert.match(
    appComponent,
    /function handleLogout\(\) \{[\s\S]*removeQueries\(\{ queryKey: \["execution-config"\] \}\)/,
  );
  assert.match(
    appComponent,
    /onSuccess=\{\(\) => \{[\s\S]*removeQueries\(\{ queryKey: \["execution-config"\] \}\)[\s\S]*setAuthenticated\(true\)/,
  );
});

test("login overlay cancels pending attempts on read-only dismissal and unmount", () => {
  const loginPage = appSource.slice(
    appSource.indexOf("function LoginPage"),
    appSource.indexOf("export default function App"),
  );
  assert.match(loginPage, /cancelPendingLoginAttempt/);
  assert.match(loginPage, /return \(\) => \{[\s\S]*cancelPendingLoginAttempt\(\)/);
  assert.match(loginPage, /function handleCancel\(\)[\s\S]*cancelPendingLoginAttempt\(\)[\s\S]*onCancel\(\)/);
  assert.match(loginPage, /nextError instanceof LoginAttemptCancelledError[\s\S]*return/);
  assert.match(loginPage, /onClick=\{handleCancel\}/);
  assert.match(loginPage, /const submitSequenceRef = useRef\(0\)/);
  assert.match(loginPage, /const submitSequence = submitSequenceRef\.current \+ 1/);
  assert.match(loginPage, /submitSequenceRef\.current === submitSequence/);
});
