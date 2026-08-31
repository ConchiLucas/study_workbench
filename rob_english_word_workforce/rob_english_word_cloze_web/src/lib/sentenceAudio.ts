export type SentenceAudioSource =
  | { kind: "minio"; url: string }
  | { kind: "speech" };

export function resolveSentenceAudioSource(sentenceAudioUrl?: string | null): SentenceAudioSource {
  const url = sentenceAudioUrl?.trim() || "";
  return url ? { kind: "minio", url } : { kind: "speech" };
}
