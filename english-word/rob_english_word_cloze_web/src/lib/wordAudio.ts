export type WordAudioSource =
  | { kind: "minio"; url: string }
  | { kind: "missing" };

export function resolveWordAudioSource(wordAudioUrl?: string | null): WordAudioSource {
  const url = wordAudioUrl?.trim() || "";
  return url ? { kind: "minio", url } : { kind: "missing" };
}

export function wordAudioButtonLabel(baseWord: string, displayedWord: string, playable: boolean): string {
  if (!playable) {
    return "暂无原词音频";
  }
  return baseWord === displayedWord ? "播放发音" : `播放原词 ${baseWord}`;
}
