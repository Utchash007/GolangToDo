package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWhere_NoFilters(t *testing.T) {
	where, args := buildWhere(TaskFilter{})
	assert.Empty(t, where)
	assert.Empty(t, args)
}

func TestBuildWhere_PriorityOnly(t *testing.T) {
	where, args := buildWhere(TaskFilter{Priority: PriorityHigh})
	assert.Equal(t, " WHERE priority = $1", where)
	require.Len(t, args, 1)
	assert.Equal(t, PriorityHigh, args[0])
}

func TestBuildWhere_CategoryOnly(t *testing.T) {
	cat := "work"
	where, args := buildWhere(TaskFilter{Category: &cat})
	assert.Equal(t, " WHERE category = $1", where)
	require.Len(t, args, 1)
	assert.Equal(t, "work", args[0])
}

func TestBuildWhere_CompletedOnly(t *testing.T) {
	done := true
	where, args := buildWhere(TaskFilter{Completed: &done})
	assert.Equal(t, " WHERE completed = $1", where)
	require.Len(t, args, 1)
	assert.Equal(t, true, args[0])
}

func TestBuildWhere_AllFilters(t *testing.T) {
	cat := "work"
	done := false
	where, args := buildWhere(TaskFilter{
		Priority:  PriorityLow,
		Category:  &cat,
		Completed: &done,
	})
	assert.Contains(t, where, "priority = $1")
	assert.Contains(t, where, "category = $2")
	assert.Contains(t, where, "completed = $3")
	assert.Contains(t, where, " AND ")
	require.Len(t, args, 3)
}

func TestBuildWhere_InvalidPrioritySkipped(t *testing.T) {
	where, args := buildWhere(TaskFilter{Priority: PriorityUnknown})
	assert.Empty(t, where)
	assert.Empty(t, args)
}
