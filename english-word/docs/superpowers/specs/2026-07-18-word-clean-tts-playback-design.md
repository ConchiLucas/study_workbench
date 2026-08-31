# Word Clean TTS Playback Design

## Goal

Pre-initialize one TTS record for every row in `public.word_clean`, allow the external task center to fill those records with MinIO audio metadata, and make the cloze web application play only that generated base-word audio for word pronunciation.

Sentence playback remains unchanged and continues to use `sentenceAudioUrl`.

## Scope

This project owns only:

1. The `word_clean_tts` schema and idempotent initialization from `word_clean`.
2. Read-only consumption of successful word TTS records in the Java cloze API.
3. Playback of `wordAudioUrl` in the cloze React application.

The task center's job claiming, concurrency control, TTS generation, retries, MinIO upload, and database update behavior are explicitly out of scope. The task center may read and update `word_clean_tts` according to its own implementation.

## Data Model

Create `public.word_clean_tts` with one record per `word_clean` row:

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | `bigint identity` | Primary key |
| `word_clean_id` | `bigint not null` | Foreign key to `word_clean.id` |
| `word` | `varchar(100) not null` | Exact base-word snapshot used for lookup and TTS text |
| `status` | `varchar(32) not null default 'pending'` | Task center generation status |
| `provider` | `varchar(64) not null default ''` | TTS provider |
| `model` | `varchar(128) not null default ''` | TTS model |
| `voice` | `varchar(128) not null default ''` | Voice name |
| `audio_format` | `varchar(32) not null default ''` | Audio format |
| `tts_bucket` | `varchar(128) not null default ''` | MinIO bucket |
| `tts_object_key` | `text not null default ''` | MinIO object key |
| `tts_object_url` | `text not null default ''` | Browser-accessible MinIO proxy URL |
| `content_type` | `varchar(128) not null default ''` | Audio content type |
| `file_size` | `bigint null` | Audio byte size |
| `duration_ms` | `integer null` | Audio duration |
| `generated_at` | `timestamptz null` | Successful generation time |
| `error_message` | `text not null default ''` | Last generation error supplied by task center |
| `created_at` | `timestamptz not null default now()` | Creation time |
| `updated_at` | `timestamptz not null default now()` | Update time |

Constraints and indexes:

- Primary key on `id`.
- Unique index on `word_clean_id` so one base word has one current TTS record.
- Unique index on `word`, matching the existing exact unique index on `word_clean.word`.
- Foreign key from `word_clean_id` to `word_clean.id` with `ON DELETE CASCADE`.
- Index on `status` for task-center reads.

No case folding or trimming is used for application lookup. The exact `word_clean.word` value is copied into `word_clean_tts.word` and later joined directly.

## Initialization

Initialization is idempotent and only inserts missing rows:

```sql
INSERT INTO public.word_clean_tts (word_clean_id, word, status)
SELECT wc.id, wc.word, 'pending'
FROM public.word_clean wc
ON CONFLICT (word_clean_id) DO NOTHING;
```

Existing TTS status and MinIO metadata are never overwritten by re-running initialization. Re-running the SQL after new words are added creates only the missing `pending` records.

After initialization:

- `word_clean_tts` count must equal `word_clean` count.
- No `word_clean` row may be missing a related TTS record.
- No duplicate `word_clean_id` or `word` may exist.

## Java API Consumption

Add `wordAudioUrl` to `ClozePracticeTaskResponse`.

When building task responses, the service collects the distinct base words from the selected tasks and performs one batch lookup in `word_clean_tts`. A record is playable only when:

```sql
status = 'success'
AND tts_object_url <> ''
```

The lookup uses exact word equality. The resulting URL map is applied to each task's `wordAudioUrl`. Missing and unsuccessful records produce a null/blank `wordAudioUrl`; the Java service never generates audio or changes `word_clean_tts` state.

This batch lookup avoids one database query per task.

## React Playback

Word and sentence audio remain separate:

- “播放发音” plays `task.wordAudioUrl` through an `Audio` element.
- “朗读完整句子” continues to play `task.sentenceAudioUrl`.
- If an answer uses an inflected form such as `values` but `task.word` is `value`, the word button plays the base word `value`.
- When the displayed answer differs from the base word, the button label becomes `播放原词 value`.
- When `wordAudioUrl` is missing, the button is disabled and displays `暂无原词音频`.

The word playback path must not use dictionary-provided audio or `SpeechSynthesis`. Existing phonetic-text lookup may remain, but any audio URL returned by the dictionary service is ignored. Sentence speech behavior is not changed by this feature.

## Error Handling

- A rejected `Audio.play()` call clears the playing state and shows a retry message.
- An audio element error reports that the MinIO word audio could not be read.
- Missing word audio does not trigger online generation, dictionary audio, or browser speech.
- Failure of word audio never prevents answering or sentence playback.

## Verification

Database verification:

- Row-count parity between `word_clean` and `word_clean_tts`.
- Zero missing foreign-key relationships and zero duplicate base words.
- Random successful records return playable bytes through their stored MinIO URLs.

Backend verification:

- Mapper/service tests cover exact-word batch lookup, `success` filtering, and missing audio.
- Response tests prove `wordAudioUrl` is independent from `sentenceAudioUrl`.

Frontend verification:

- Tests prove word playback uses `wordAudioUrl`.
- Tests prove missing audio disables the button without invoking browser speech.
- Tests prove sentence playback still uses `sentenceAudioUrl`.
- Production build and browser smoke testing complete without console errors.
