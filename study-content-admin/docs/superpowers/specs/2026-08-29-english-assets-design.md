# English assets (字图 / 义图 / TTS) — design

Date: 2026-08-29  
Status: approved

## Goal

Content-admin menu **英语** (`/english`), mirroring **识字**: sync 200 vocabulary KPs from workbench subject `english`, manage **glyph (spelling card)**, **sense image**, and **EN TTS**.

## Scope

| Item | Decision |
|------|----------|
| Source | `subjects.code = 'english'` — 20 modules × 10 words |
| Glyph | Plain white PNG of the **English word spelling** (no 田字格; auto-fit long words) |
| Sense | Same sticker style as literacy; subject from English word meaning map; no letters on image |
| Skip sense | Abstract greetings / yes-no style words (`hello`, `hi`, `bye`, `thanks`, `please`, `sorry`, `yes`, `no`, `ok`, …) + override via PATCH |
| TTS | Speak the word with Grok TTS `language=en` |
| UI | Browse / play / toggle 要义图; no generate buttons (batch via API / agent) |

## Data

Table `english_assets` (parallel to `literacy_assets`):

- `kp_id`, `word_text`, module fields, `needs_sense_image` + override
- `glyph_image_url`, `sense_image_url`, `speech_audio_url`
- MinIO keys: `english/glyphs/{id}.png`, `english/senses/{id}.png`, `english/speech/{id}.mp3`

## API

Under `/api/v1/english/`:

- `POST /sync`, `GET /words?view=groups|table`
- `PATCH /words/:kpId` (sense override)
- Glyph: `POST …/glyph`, `GET …/glyph.png`, `POST /glyphs/batch`
- Sense: `POST …/sense`, `GET …/sense.png`, `POST /senses/batch`
- Speech: `GET …/speech.mp3` (lazy cache), `POST …/speech`, `POST /speech/batch?moduleCode=`

## Frontend

- Nav **英语**, route `/english`, page shaped like `LiteracyPage`
- Fields use `wordText` instead of `charText`

## Out of scope

- Wiring kid-app quiz to replace emoji with sense PNGs (follow-up)
- Phrase subject (`phrase` / 英语短句)
