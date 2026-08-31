package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-workbench/internal/db"
)

func TestMigrateCreatesTables(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))

	for _, table := range []string{
		"users", "children", "parent_child", "subjects", "modules",
		"knowledge_points", "questions", "attempts", "mastery_states",
		"study_sessions", "daily_stats", "daily_tasks", "rewards", "flower_ledger",
	} {
		require.True(t, gdb.Migrator().HasTable(table), "缺少表 %s", table)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, db.Migrate(gdb))
}
