import type {
  CLIDriver,
  CLIProviderConfigItem,
  CodexReasoningEffort,
  ExecutionAIProviderConfigItem,
  ExecutionConfig,
  ExecutionTarget,
} from "../types/executionConfig.ts";

interface SelectOption<T extends string> {
  label: T;
  value: T;
}

const CODEX_MODELS = [
  "gpt-5.6-sol",
  "gpt-5.6-terra",
  "gpt-5.6-luna",
  "gpt-5.5",
  "gpt-5.4",
  "gpt-5.4-mini",
  "gpt-5.3-codex-spark",
] as const;

const GEMINI_MODELS = ["auto", "pro", "flash", "flash-lite"] as const;

const CODEX_REASONING_EFFORTS = ["low", "medium", "high", "xhigh"] as const;

export const CLI_MODEL_OPTIONS = {
  codex: CODEX_MODELS.map((value) => ({ label: value, value })),
  gemini: GEMINI_MODELS.map((value) => ({ label: value, value })),
} satisfies Record<CLIDriver, ReadonlyArray<SelectOption<string>>>;

export const CLI_REASONING_OPTIONS = CODEX_REASONING_EFFORTS.map((value) => ({
  label: value,
  value,
})) satisfies ReadonlyArray<SelectOption<CodexReasoningEffort>>;

export const DEFAULT_WORKING_DIRECTORY = "/Users/conchi/workforce/rob_english_word_workforce";

export function isGoSpaceCodePoint(codePoint: number): boolean {
  return codePoint >= 0x09 && codePoint <= 0x0d
    || codePoint === 0x20
    || codePoint === 0x85
    || codePoint === 0xa0
    || codePoint === 0x1680
    || codePoint >= 0x2000 && codePoint <= 0x200a
    || codePoint === 0x2028
    || codePoint === 0x2029
    || codePoint === 0x202f
    || codePoint === 0x205f
    || codePoint === 0x3000;
}

type GoSpacePredicate = (codePoint: number) => boolean;

export function findGoTrimBounds(
  value: string,
  isSpace: GoSpacePredicate = isGoSpaceCodePoint,
): [number, number] {
  let start = 0;
  while (start < value.length && isSpace(value.charCodeAt(start))) {
    start += 1;
  }

  let end = value.length;
  while (end > start && isSpace(value.charCodeAt(end - 1))) {
    end -= 1;
  }
  return [start, end];
}

// Go strings.TrimSpace uses Unicode White Space, not JavaScript's WhiteSpace
// production (which also includes U+FEFF). All Go spaces are in the BMP, so a
// two-ended code-unit scan is stable and linear without inspecting the middle.
export function trimLikeGo(value: string): string {
  const [start, end] = findGoTrimBounds(value);
  return value.slice(start, end);
}

const DEFAULT_CLI_PROVIDERS: Record<CLIDriver, Omit<CLIProviderConfigItem, "id">> = {
  codex: {
    label: "Codex CLI",
    driver: "codex",
    command_path: "/Applications/ChatGPT.app/Contents/Resources/codex",
    model: "gpt-5.6-sol",
    reasoning_effort: "high",
    working_directory: DEFAULT_WORKING_DIRECTORY,
    timeout_seconds: 300,
    enabled: true,
  },
  gemini: {
    label: "Gemini CLI",
    driver: "gemini",
    command_path: "/Users/conchi/.npm-global/bin/gemini",
    model: "auto",
    reasoning_effort: "",
    working_directory: DEFAULT_WORKING_DIRECTORY,
    timeout_seconds: 300,
    enabled: true,
  },
};

function uniqueProviderID(baseID: string, existingIDs: Iterable<string>): string {
  const used = new Set(Array.from(existingIDs, (id) => trimLikeGo(id)));
  if (!used.has(baseID)) {
    return baseID;
  }
  let suffix = 2;
  while (used.has(`${baseID}-${suffix}`)) {
    suffix += 1;
  }
  return `${baseID}-${suffix}`;
}

export function createDefaultCLIProvider(
  driver: CLIDriver,
  existingIDs: Iterable<string> = [],
): CLIProviderConfigItem {
  return {
    id: uniqueProviderID(driver, existingIDs),
    ...DEFAULT_CLI_PROVIDERS[driver],
  };
}

export function createDefaultExecutionConfig(): ExecutionConfig {
  return {
    active_target: null,
    api_providers: [],
    cli_providers: [createDefaultCLIProvider("codex"), createDefaultCLIProvider("gemini")],
  };
}

function isHexDigit(value: string): boolean {
  return /^[0-9a-f]$/i.test(value);
}

function hasValidPercentEscapes(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    if (value[index] !== "%") {
      continue;
    }
    if (
      index + 2 >= value.length
      || !isHexDigit(value[index + 1])
      || !isHexDigit(value[index + 2])
    ) {
      return false;
    }
    index += 2;
  }
  return true;
}

function isAllowedHostASCII(value: string): boolean {
  return /^[a-z0-9\-._~!$&'()*+,;=:]$/i.test(value);
}

function decodeUTF8LikeGo(bytes: number[]): string {
  const isContinuation = (value: number): boolean => value >= 0x80 && value <= 0xbf;
  let result = "";
  for (let index = 0; index < bytes.length;) {
    const first = bytes[index];
    if (first <= 0x7f) {
      result += String.fromCodePoint(first);
      index += 1;
      continue;
    }

    const second = bytes[index + 1];
    if (first >= 0xc2 && first <= 0xdf && isContinuation(second)) {
      result += String.fromCodePoint(((first & 0x1f) << 6) | (second & 0x3f));
      index += 2;
      continue;
    }

    const third = bytes[index + 2];
    const validThreeByteSecond = (
      first === 0xe0 && second >= 0xa0 && second <= 0xbf
      || first >= 0xe1 && first <= 0xec && isContinuation(second)
      || first === 0xed && second >= 0x80 && second <= 0x9f
      || first >= 0xee && first <= 0xef && isContinuation(second)
    );
    if (validThreeByteSecond && isContinuation(third)) {
      result += String.fromCodePoint(
        ((first & 0x0f) << 12) | ((second & 0x3f) << 6) | (third & 0x3f),
      );
      index += 3;
      continue;
    }

    const fourth = bytes[index + 3];
    const validFourByteSecond = (
      first === 0xf0 && second >= 0x90 && second <= 0xbf
      || first >= 0xf1 && first <= 0xf3 && isContinuation(second)
      || first === 0xf4 && second >= 0x80 && second <= 0x8f
    );
    if (validFourByteSecond && isContinuation(third) && isContinuation(fourth)) {
      result += String.fromCodePoint(
        ((first & 0x07) << 18)
        | ((second & 0x3f) << 12)
        | ((third & 0x3f) << 6)
        | (fourth & 0x3f),
      );
      index += 4;
      continue;
    }

    // JavaScript strings cannot contain Go's raw invalid UTF-8 bytes. Map each
    // one to a distinct low-surrogate sentinel so origin equality stays lossless
    // and cannot collide with a valid U+FFFD replacement character.
    result += String.fromCharCode(0xdc00 + first);
    index += 1;
  }
  return result;
}

function decodePercentBytes(value: string): string {
  let result = "";
  for (let index = 0; index < value.length;) {
    if (value[index] !== "%") {
      result += value[index];
      index += 1;
      continue;
    }
    const bytes: number[] = [];
    while (value[index] === "%") {
      bytes.push(Number.parseInt(value.slice(index + 1, index + 3), 16));
      index += 3;
    }
    result += decodeUTF8LikeGo(bytes);
  }
  return result;
}

function lowercaseASCIIHostname(value: string): string {
  return value.replace(/[A-Z]/g, (character) => (
    String.fromCharCode(character.charCodeAt(0) + ("a".charCodeAt(0) - "A".charCodeAt(0)))
  ));
}

function hasUnpairedSurrogate(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const codeUnit = value.charCodeAt(index);
    if (codeUnit >= 0xd800 && codeUnit <= 0xdbff) {
      if (index + 1 >= value.length) {
        return true;
      }
      const next = value.charCodeAt(index + 1);
      if (next < 0xdc00 || next > 0xdfff) {
        return true;
      }
      index += 1;
    } else if (codeUnit >= 0xdc00 && codeUnit <= 0xdfff) {
      return true;
    }
  }
  return false;
}

function decodeGoHostPart(value: string, zone: boolean): string | null {
  if (!hasValidPercentEscapes(value)) {
    return null;
  }
  for (let index = 0; index < value.length; index += 1) {
    const character = value[index];
    if (character === "%") {
      const encoded = value.slice(index, index + 3);
      const decodedByte = Number.parseInt(encoded.slice(1), 16);
      if (zone && decodedByte >= 0x80) {
        return null;
      }
      if (
        !zone && decodedByte < 0x80 && encoded !== "%25"
        || zone
          && decodedByte < 0x80
          && decodedByte !== 0x25
          && decodedByte !== 0x20
          && decodedByte !== 0x22
          && decodedByte !== 0x3c
          && decodedByte !== 0x3e
          && decodedByte !== 0x5b
          && decodedByte !== 0x5d
          && !isAllowedHostASCII(String.fromCharCode(decodedByte))
      ) {
        return null;
      }
      index += 2;
      continue;
    }
    const codePoint = character.codePointAt(0) ?? 0;
    const extraGoZoneCharacter = zone && (character === "\"" || character === "<" || character === ">");
    if (codePoint < 0x80 && !isAllowedHostASCII(character) && !extraGoZoneCharacter) {
      return null;
    }
  }
  return decodePercentBytes(value);
}

function isValidIPv4Tail(value: string): boolean {
  const parts = value.split(".");
  return parts.length === 4 && parts.every((part) => (
    /^(0|[1-9][0-9]{0,2})$/.test(part) && Number(part) <= 255
  ));
}

function countIPv6Groups(value: string, allowIPv4Tail: boolean): number | null {
  if (!value) {
    return 0;
  }
  const parts = value.split(":");
  if (parts.some((part) => !part)) {
    return null;
  }
  let groups = 0;
  for (let index = 0; index < parts.length; index += 1) {
    const part = parts[index];
    if (part.includes(".")) {
      if (!allowIPv4Tail || index !== parts.length - 1 || !isValidIPv4Tail(part)) {
        return null;
      }
      groups += 2;
    } else if (/^[0-9a-f]{1,4}$/i.test(part)) {
      groups += 1;
    } else {
      return null;
    }
  }
  return groups;
}

function isValidBracketedIPv6(value: string): boolean {
  const zoneSeparator = value.indexOf("%");
  const address = zoneSeparator < 0 ? value : value.slice(0, zoneSeparator);
  if (zoneSeparator >= 0 && !value.slice(zoneSeparator + 1)) {
    return false;
  }
  const compressedParts = address.split("::");
  if (compressedParts.length > 2) {
    return false;
  }
  const leftGroups = countIPv6Groups(compressedParts[0], compressedParts.length === 1);
  const rightGroups = countIPv6Groups(
    compressedParts.length === 2 ? compressedParts[1] : "",
    true,
  );
  if (leftGroups === null || rightGroups === null) {
    return false;
  }
  const groupCount = leftGroups + rightGroups;
  return compressedParts.length === 2 ? groupCount < 8 : groupCount === 8;
}

interface ParsedExecutionOriginAuthority {
  hostname: string;
  port: string;
}

function parseExecutionOriginAuthority(authority: string): ParsedExecutionOriginAuthority | null {
  if (!authority || authority.includes("@")) {
    return null;
  }

  if (authority.startsWith("[")) {
    const closingBracket = authority.lastIndexOf("]");
    if (closingBracket <= 1 || authority.slice(1, closingBracket).includes("[")) {
      return null;
    }
    const portPart = authority.slice(closingBracket + 1);
    if (portPart && !/^:[0-9]*$/.test(portPart)) {
      return null;
    }
    const rawHostname = authority.slice(1, closingBracket);
    const zoneSeparator = rawHostname.indexOf("%25");
    const hostPart = zoneSeparator < 0 ? rawHostname : rawHostname.slice(0, zoneSeparator);
    const zonePart = zoneSeparator < 0 ? "" : rawHostname.slice(zoneSeparator);
    const decodedHost = decodeGoHostPart(hostPart, false);
    const decodedZone = zonePart ? decodeGoHostPart(zonePart, true) : "";
    if (decodedHost === null || decodedZone === null) {
      return null;
    }
    const hostname = decodedHost + decodedZone;
    if (!isValidBracketedIPv6(hostname)) {
      return null;
    }
    return { hostname, port: portPart.slice(1) };
  }

  if (authority.includes("[") || authority.includes("]")) {
    return null;
  }
  // The backend module targets Go 1.25. Under Go 1.26 this keeps the
  // urlstrictcolons=0 compatibility behavior and splits on the last colon.
  const portSeparator = authority.lastIndexOf(":");
  const rawHostname = portSeparator < 0 ? authority : authority.slice(0, portSeparator);
  const portPart = portSeparator < 0 ? "" : authority.slice(portSeparator);
  if (portPart && !/^:[0-9]*$/.test(portPart)) {
    return null;
  }
  const hostname = decodeGoHostPart(rawHostname, false);
  if (!hostname) {
    return null;
  }
  return { hostname, port: portPart.slice(1) };
}

export function normalizedExecutionAIOrigin(raw: string): string | null {
  const value = trimLikeGo(raw);
  if (!value || /[\u0000-\u001f\u007f]/.test(value) || hasUnpairedSurrogate(value)) {
    return null;
  }

  const fragmentSeparator = value.indexOf("#");
  const fragment = fragmentSeparator < 0 ? "" : value.slice(fragmentSeparator + 1);
  if (fragment) {
    return null;
  }
  const withoutFragment = fragmentSeparator < 0 ? value : value.slice(0, fragmentSeparator);
  const querySeparator = withoutFragment.indexOf("?");
  const rawQuery = querySeparator < 0 ? "" : withoutFragment.slice(querySeparator + 1);
  if (rawQuery) {
    return null;
  }
  const withoutQuery = querySeparator < 0
    ? withoutFragment
    : withoutFragment.slice(0, querySeparator);

  const schemeMatch = /^([a-z][a-z0-9+.-]*):\/\//i.exec(withoutQuery);
  if (!schemeMatch) {
    return null;
  }
  const scheme = schemeMatch[1].toLowerCase();
  if (scheme !== "http" && scheme !== "https") {
    return null;
  }

  const afterScheme = withoutQuery.slice(schemeMatch[0].length);
  const pathSeparator = afterScheme.indexOf("/");
  const authority = pathSeparator < 0 ? afterScheme : afterScheme.slice(0, pathSeparator);
  const path = pathSeparator < 0 ? "" : afterScheme.slice(pathSeparator);
  if (!hasValidPercentEscapes(path)) {
    return null;
  }
  const parsedAuthority = parseExecutionOriginAuthority(authority);
  if (!parsedAuthority) {
    return null;
  }
  const port = parsedAuthority.port || (scheme === "https" ? "443" : "80");
  return `${scheme}://${lowercaseASCIIHostname(parsedAuthority.hostname)}:${port}`;
}

export function applyExecutionAIProviderEdit(
  provider: ExecutionAIProviderConfigItem,
  patch: Partial<ExecutionAIProviderConfigItem>,
): ExecutionAIProviderConfigItem {
  let apiKeyConfigured = provider.api_key_configured;
  if (patch.id !== undefined && trimLikeGo(patch.id) !== trimLikeGo(provider.id)) {
    apiKeyConfigured = false;
  }
  if (patch.base_url !== undefined) {
    const previousOrigin = normalizedExecutionAIOrigin(provider.base_url);
    const nextOrigin = normalizedExecutionAIOrigin(patch.base_url);
    if (!previousOrigin || !nextOrigin || previousOrigin !== nextOrigin) {
      apiKeyConfigured = false;
    }
  }
  return { ...provider, ...patch, api_key_configured: apiKeyConfigured };
}

function validateAPIProviders(providers: ExecutionAIProviderConfigItem[]): string | null {
  const ids = new Set<string>();
  for (const provider of providers) {
    const id = trimLikeGo(provider.id);
    if (!id) {
      return "API provider ID 不能为空";
    }
    if (ids.has(id)) {
      return `API provider ID「${id}」重复`;
    }
    ids.add(id);
    if (!trimLikeGo(provider.label)) {
      return `API provider「${id}」的 label 不能为空`;
    }
    if (!trimLikeGo(provider.type)) {
      return `API provider「${id}」的 type 不能为空`;
    }
    if (!trimLikeGo(provider.base_url) || !normalizedExecutionAIOrigin(provider.base_url)) {
      return `API provider「${id}」的 base_url 无效`;
    }
    if (!trimLikeGo(provider.model)) {
      return `API provider「${id}」的 model 不能为空`;
    }
    if (!Number.isSafeInteger(provider.max_tokens) || provider.max_tokens <= 0) {
      return `API provider「${id}」的 max_tokens 必须大于 0`;
    }
    if (!provider.api_key_configured && !trimLikeGo(provider.api_key)) {
      return `请填写 API provider「${id}」的 API Key`;
    }
  }
  return null;
}

function validateCLIProviders(providers: CLIProviderConfigItem[]): string | null {
  const ids = new Set<string>();
  const codexModels = new Set<string>(CODEX_MODELS);
  const geminiModels = new Set<string>(GEMINI_MODELS);
  const codexReasoning = new Set<string>(CODEX_REASONING_EFFORTS);

  for (const provider of providers) {
    const id = trimLikeGo(provider.id);
    if (!id) {
      return "CLI provider ID 不能为空";
    }
    if (ids.has(id)) {
      return `CLI provider ID「${id}」重复`;
    }
    ids.add(id);
    if (!trimLikeGo(provider.label)) {
      return `CLI provider「${id}」的 label 不能为空`;
    }
    const driver = trimLikeGo(provider.driver);
    if (driver !== "codex" && driver !== "gemini") {
      return `CLI provider「${id}」的 driver 仅支持 codex 或 gemini`;
    }
    if (!trimLikeGo(provider.command_path).startsWith("/")) {
      return `CLI provider「${id}」的 command_path 必须是非空绝对路径`;
    }
    if (!trimLikeGo(provider.working_directory).startsWith("/")) {
      return `CLI provider「${id}」的 working_directory 必须是非空绝对路径`;
    }
    if (!Number.isSafeInteger(provider.timeout_seconds) || provider.timeout_seconds <= 0) {
      return `CLI provider「${id}」的 timeout_seconds 必须大于 0`;
    }

    const model = trimLikeGo(provider.model);
    const reasoning = trimLikeGo(provider.reasoning_effort);
    if (driver === "codex") {
      if (!codexModels.has(model)) {
        return `CLI provider「${id}」的 model 不受支持`;
      }
      if (!codexReasoning.has(reasoning)) {
        return `CLI provider「${id}」的 reasoning_effort 不受支持`;
      }
    } else {
      if (!geminiModels.has(model)) {
        return `CLI provider「${id}」的 model 不受支持`;
      }
      if (reasoning !== "") {
        return `CLI provider「${id}」的 Gemini reasoning_effort 必须为空`;
      }
    }
  }
  return null;
}

export function validateExecutionConfig(config: ExecutionConfig): string | null {
  const apiError = validateAPIProviders(config.api_providers);
  if (apiError) {
    return apiError;
  }
  const cliError = validateCLIProviders(config.cli_providers);
  if (cliError) {
    return cliError;
  }

  const target = config.active_target as ExecutionTarget | null;
  if (!target) {
    return "请选择造句执行器";
  }
  if (typeof target.type !== "string") {
    return "造句执行器类型无效";
  }
  const targetType = trimLikeGo(target.type);
  if (targetType !== "api" && targetType !== "cli") {
    return "造句执行器类型无效";
  }
  if (typeof target.id !== "string") {
    return "造句执行器 ID 无效";
  }
  const targetID = trimLikeGo(target.id);
  if (!targetID) {
    return "造句执行器 ID 不能为空";
  }
  if (targetType === "api") {
    if (!config.api_providers.some((provider) => trimLikeGo(provider.id) === targetID)) {
      return `API 造句执行器「${targetID}」不存在`;
    }
    return null;
  }

  const provider = config.cli_providers.find((item) => trimLikeGo(item.id) === targetID);
  if (!provider) {
    return `CLI 造句执行器「${targetID}」不存在`;
  }
  if (!provider.enabled) {
    return `CLI 造句执行器「${targetID}」已停用`;
  }
  return null;
}

export function canDeleteAPI(config: ExecutionConfig, id: string): boolean {
  const normalizedID = trimLikeGo(id);
  return config.api_providers.some((provider) => trimLikeGo(provider.id) === normalizedID)
    && !(
      config.active_target
      && trimLikeGo(config.active_target.type) === "api"
      && trimLikeGo(config.active_target.id) === normalizedID
    );
}

export function canDeleteCLI(config: ExecutionConfig, id: string): boolean {
  const normalizedID = trimLikeGo(id);
  return config.cli_providers.some((provider) => trimLikeGo(provider.id) === normalizedID)
    && !(
      config.active_target
      && trimLikeGo(config.active_target.type) === "cli"
      && trimLikeGo(config.active_target.id) === normalizedID
    );
}
