import { resolveWordAudioSource, wordAudioButtonLabel } from "../lib/wordAudio";

interface WordAudioButtonProps {
  baseWord: string;
  displayedWord: string;
  audioUrl?: string | null;
  onPlay: (audioUrl: string) => void;
  compact?: boolean;
}

export function WordAudioButton({
  baseWord,
  displayedWord,
  audioUrl,
  onPlay,
  compact = false,
}: WordAudioButtonProps) {
  const source = resolveWordAudioSource(audioUrl);
  const playable = source.kind === "minio";
  const label = wordAudioButtonLabel(baseWord, displayedWord, playable);

  return (
    <button
      type="button"
      className={compact ? "audio-play-btn-small" : "audio-play-btn"}
      disabled={!playable}
      onClick={() => {
        if (source.kind === "minio") {
          onPlay(source.url);
        }
      }}
      title={label}
      aria-label={label}
    >
      {compact ? "🔊" : `🔊 ${label}`}
    </button>
  );
}
