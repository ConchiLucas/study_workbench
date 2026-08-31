#!/usr/bin/env bash
# 一键刷新端到端 demo：新学科掌握度 + 任务内容 + 补做日期 + 今日计划。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PG="${PG:-docker exec -i postgres16 psql -U conchi -d study_workbench}"
API="${API:-http://localhost:8080/api/v1}"

run_sql() {
  echo ">> $1"
  $PG < "$ROOT/scripts/$1"
}

run_sql seed_chengyu_phrase_mastery.sql
run_sql diversify_demo_plans.sql
run_sql refresh_kid_catchup_demo.sql

echo ">> GET /children/1/plans/todo"
curl -sf "$API/children/1/plans/todo" | python3 -c "
import json,sys
rows=json.load(sys.stdin)['data']
print('todo tasks:', len(rows))
for r in rows:
    subs=', '.join(f\"{s['icon']}{s['count']}\" for s in r.get('subjects',[]))
    print(' ', r['plan_date'], r['status'], f\"{r['done_count']}/{r['target_count']}\", subs)
"

echo ">> GET /children/1/subjects (counts)"
curl -sf "$API/children/1/subjects" | python3 -c "
import json,sys
rows=json.load(sys.stdin)['data']
for s in rows:
    if s['code'] in ('chengyu','phrase'):
        c=s['counts']
        print(s['icon'], s['name'], c)
"

echo "done."
