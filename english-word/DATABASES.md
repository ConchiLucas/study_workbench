# Database Inventory

This file is a quick reference for the databases used in this workspace.

## Project Map

- `rob_english_word_back`: Java 后端，负责英语抢词游戏的登录、房间、对战、结算、Redis 状态和 PostgreSQL 落库。
- `rob_english_word_front`: Vue 前端，对应英语抢词游戏的用户界面。
- `rob_english_word_cloze_web`: React 前端，做挖空练习，支持登录、取题、提交答案、统计和历史记录。
- `word_select_dashboard/server`: Go 后端/管理服务，负责执行流、数据库管理、AI 配置、词句生成和记录查询。
- `word_select_dashboard/web-react`: React 管理前端，展示和管理执行流、句子生成和相关业务记录。
- `word_select_dashboard/word-agent`: Python 代理服务，接收 Go 任务、生成句子并回传步骤事件，同时写入挖空练习相关数据。

## Port Map

- `rob_english_word_back`: HTTP `8019`, WebSocket `9091`, PostgreSQL `5432`, Redis `6379`
- `rob_english_word_front`: dev `7002`, Docker web `6111 -> 80`
- `rob_english_word_cloze_web`: dev `7003`
- `word_select_dashboard/server`: API `21417`, Docker `10008 -> 21417`
- `word_select_dashboard/web-react`: dev `7001`
- `word_select_dashboard/word-agent`: `8010`
- `word_select_dashboard` shared services: PostgreSQL `5432`, Redis `6379`

## `rob_english_word_back`

- Database type: PostgreSQL
- Default database name: `rob_english_word`
- Default host/port: `127.0.0.1:5432`
- Default username: `rob_word`
- Default password: empty in local config
- Docker default database name: `rob_english_word`
- Docker default password: `change-me`

Key config files:

- [rob_english_word_back/src/main/resources/application.yml](/Users/conchi/workforce/rob_english_word_workforce/rob_english_word_back/src/main/resources/application.yml)
- [rob_english_word_back/docker-compose.yml](/Users/conchi/workforce/rob_english_word_workforce/rob_english_word_back/docker-compose.yml)
- [docker-compose.yml](/Users/conchi/workforce/rob_english_word_workforce/docker-compose.yml)

Tables from `rob_english_word_back/db`:

- `users`
- `word`
- `word_clean`: deduplicated single-word table derived from `word`, keeps `word`, `meaning`, `difficulty`, `frequency`, `sentence`, and PEP textbook difficulty fields `pep_difficulty` / `pep_difficulty_label`
  - PEP textbook filter strategy: scan from `人教版小学英语3年级上册` upward by `pep_difficulty`; if a word appears in multiple PEP levels, keep it only in the lowest level and exclude it from all higher-level filters.
  - Unified source difficulty fields: `source_difficulty` / `source_difficulty_label` extend the same lowest-level strategy beyond PEP. Current order: PEP `1-24`, CET4 `25`, KaoYan `26`, BEC `27`, CET6 `28`, IELTS `29`, TEM4 `30`, TEM8 `31`, TOEFL `32`, GMAT `33`, SAT `34`, GRE `35`, other sources `36` (`其他词库`).
- `word_clean_sentence`: generated practice sentences for `word_clean`, stores `word_clean_id`, `word`, `model_name`, and `sentence`
- `word_clean_sentence_job`: per-word sentence generation task state table for `word_clean`, stores model, status, retry count, error, and lock time
- `word_library`: includes `library_meaning`, a Chinese explanation derived from `library_name`
- `room`
- `game_record`
- `game_answer_detail`
- `user_word_state`
- `sentence_cloze_item`
- `sentence_cloze_answer_record`

Notes:

- The SQL files are schema/DDL scripts, not exported data dumps.
- `game_anwser_detail.sql` is misspelled in the filename, but the table is `game_answer_detail`.

## `word_select_dashboard`

- Database type: PostgreSQL by default
- Default database name: `select_english_word`
- Default host/port: `127.0.0.1:5432` in local config, `host.docker.internal:5432` in docker config
- Default username: `conchi` locally, `change-me-user` in docker template
- Default password: `conchi123456` locally, `change-me-password` in docker template

Key config files:

- [word_select_dashboard/server/config.yaml](/Users/conchi/workforce/rob_english_word_workforce/word_select_dashboard/server/config.yaml)
- [word_select_dashboard/server/config.docker.yaml](/Users/conchi/workforce/rob_english_word_workforce/word_select_dashboard/server/config.docker.yaml)
- [word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py](/Users/conchi/workforce/rob_english_word_workforce/word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py)

Observed table usage:

- `wrong_word_events` is created on demand by `word-agent`
- The dashboard code reads/writes several PostgreSQL-backed system tables through GORM, but I did not find a separate SQL dump folder like `rob_english_word_back/db`

## Redis

- `rob_english_word_back` uses Redis at `127.0.0.1:6379` by default
- `word_select_dashboard` also has Redis config, with local and docker variants

## Service Restart Scripts

## `word_clean` Sentence Generation

- Management page: `word_select_dashboard/web-react`, left menu `词库管理 -> 造句任务表`
- Start endpoint through Go: `POST /word-libraries/clean-sentence-jobs/run`
- Go forwards the start request to `word-agent`: `POST /v1/word-clean-sentences/run`
- Normal resume request uses `failedOnly=false`: initializes missing jobs, then runs only `pending` tasks and stale `running` tasks.
- Failed retry request uses `failedOnly=true`: skips initialization and runs only `failed` tasks whose `retry_count` is still below the retry limit.
- Default batch size: `20`
- Default max retries per word/model: `3`
- Default model order: `deepseek-v3.2`, `qwen3.6-flash`, `kimi-k2.5`, `glm-5`, `MiniMax-M2.5`
- The Python agent initializes `word_clean_sentence_job` from `word_clean` for one model, runs all eligible jobs for that model, then moves to the next model.
- Each batch calls the model once with up to 20 words, but validates and writes results per `word_clean_id`.
- Successful rows are upserted into `word_clean_sentence`; failed rows stay in `word_clean_sentence_job` with `status='failed'`, `retry_count`, and `error_message`.
- Do not start the full generation endpoint during verification unless the user explicitly wants to spend model calls and run the full job.

When the user only asks to start or restart the workspace, do not read this file. Read `START.md` only and run:

```bash
./restart_all_services.sh restart
```

When changing code or config for any service in this workspace:

1. Read this `DATABASES.md` file first to confirm the project, port, database, and matching restart script.
2. Make the requested change.
3. Restart only changed backend services with their own script from that service directory.
4. Do not restart frontend services after ordinary frontend code/style changes; the dev server should hot reload them.
5. Restart a frontend service only when adding or changing dependencies, or when the user explicitly asks for a frontend restart.
6. Use `restart_all_services.sh` only when the request explicitly needs the whole workspace restarted.

Per-service restart scripts:

- `rob_english_word_back`: `rob_english_word_back/restart_rob_english_word_back.sh`
- `rob_english_word_front`: `rob_english_word_front/restart_rob_english_word_front.sh`
- `rob_english_word_cloze_web`: `rob_english_word_cloze_web/restart_rob_english_word_cloze_web.sh`
- `word_select_dashboard/server`: `word_select_dashboard/server/restart_word_select_dashboard_server.sh`
- `word_select_dashboard/web-react`: `word_select_dashboard/web-react/restart_word_select_dashboard_web.sh`
- `word_select_dashboard/word-agent`: `word_select_dashboard/word-agent/restart_word_agent.sh`

The scripts share common PID, log, port-cleanup, and launchd logic from `restart_service_common.sh`.
Logs and PID files are written under `.service-runtime/`.

## Quick Read Path

When you ask me to read table data later, the fastest path is:

1. Check which project you mean.
2. Open the matching config file above.
3. Read the schema files or GORM model for the table.

## Database Read Commands

The local shell does not have `psql`, but `word_select_dashboard/word-agent/.venv` has `psycopg`.
Use that Python environment to run read-only PostgreSQL queries.

Read `rob_english_word` locally:

```bash
word_select_dashboard/word-agent/.venv/bin/python -c "import psycopg; conn=psycopg.connect('host=127.0.0.1 port=5432 dbname=rob_english_word user=conchi password=conchi123456'); cur=conn.execute('select id, library_name, status, word_count, created_by from public.word_library order by id'); print('\n'.join(str(row) for row in cur.fetchall())); conn.close()"
```

Notes:

- Sandbox may block local TCP access to `127.0.0.1:5432`; if it says `Operation not permitted`, rerun the same read-only command with approval outside the sandbox.
- The working credentials observed in this workspace are `user=conchi password=conchi123456` for local `rob_english_word`.
- The Docker/default credentials in compose files may differ, such as `user=rob_word password=change-me`.

## Which File To Open

- Ask about `rob_english_word_back` tables: start from [rob_english_word_back/db](/Users/conchi/workforce/rob_english_word_workforce/rob_english_word_back/db)
- Ask about `word_select_dashboard` PostgreSQL data: start from [word_select_dashboard/server/config.yaml](/Users/conchi/workforce/rob_english_word_workforce/word_select_dashboard/server/config.yaml) and the matching GORM models in [word_select_dashboard/server/model](/Users/conchi/workforce/rob_english_word_workforce/word_select_dashboard/server/model)
- Ask about `word-agent` generated data: start from [word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py](/Users/conchi/workforce/rob_english_word_workforce/word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py)
- Ask about cloze front-end behavior: start from [rob_english_word_cloze_web/src/App.tsx](/Users/conchi/workforce/rob_english_word_workforce/rob_english_word_cloze_web/src/App.tsx)
