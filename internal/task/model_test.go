package task

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTask_ID(t *testing.T) {
	task := NewTask("Test Task", PriorityMedium, "work")

	assert.NotEqual(t, uuid.Nil, task.ID, "task ID should be generated")
}

func TestTask_Timestamps(t *testing.T) {
	now := time.Now()
	task := NewTask("Test Task", PriorityMedium, "work")

	assert.WithinDuration(t, now, task.CreatedAt, time.Second, "created_at should be set")
	assert.WithinDuration(t, now, task.UpdatedAt, time.Second, "updated_at should be set")
}

func TestPriority_String(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityMedium, "medium"},
		{PriorityHigh, "high"},
		{PriorityUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.priority.String())
		})
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected Priority
	}{
		{"low", PriorityLow},
		{"medium", PriorityMedium},
		{"high", PriorityHigh},
		{"invalid", PriorityUnknown},
		{"", PriorityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParsePriority(tt.input))
		})
	}
}

func TestPriority_IsValid(t *testing.T) {
	assert.True(t, PriorityLow.IsValid())
	assert.True(t, PriorityMedium.IsValid())
	assert.True(t, PriorityHigh.IsValid())
	assert.False(t, PriorityUnknown.IsValid())
}