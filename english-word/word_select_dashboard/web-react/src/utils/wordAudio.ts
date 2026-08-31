export function playableTTSAudioURL(status: string, objectURL: string): string | null {
  const audioURL = objectURL.trim();
  return status === "success" && audioURL ? audioURL : null;
}

export const playableBestSentenceAudioURL = playableTTSAudioURL;
