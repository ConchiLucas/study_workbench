# Word Agent TTS example

This example calls the local Word Agent TTS endpoint and saves the returned
audio as a local `.wav` file. Start Word Agent first and configure Xiaomi MiMo
under the dashboard's **TTS 模型配置** page.

```bash
python3 scripts/mimo_tts_word.py apple
```

The default output is:

```text
tts_audio/apple.wav
```

Useful options:

```bash
python3 scripts/mimo_tts_word.py apple -o tts_audio/apple_custom.wav
python3 scripts/mimo_tts_word.py apple --word-agent-url http://127.0.0.1:6017
python3 scripts/mimo_tts_word.py "good morning" --style "Warm English teacher voice. Speak slowly."
```

The script does not read provider credentials, model names, or voice names.
Those values come from the active database-backed TTS configuration used by
Word Agent. Its request flow is:

1. `POST /v1/tts/generate`
2. Read `downloadUrl` from the JSON response
3. Download the WAV from Word Agent
