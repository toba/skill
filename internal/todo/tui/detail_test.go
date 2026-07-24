package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/toba/jig/internal/todo/issue"
)

func TestRenderDates(t *testing.T) {
	created := time.Date(2026, 7, 20, 9, 30, 0, 0, time.UTC)
	updated := time.Date(2026, 7, 24, 12, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		created *time.Time
		updated *time.Time
		want    []string // substrings that must be present ("" means output must be empty)
		empty   bool
	}{
		{
			name:    "both dates",
			created: &created,
			updated: &updated,
			want:    []string{"created 2026-07-20", "updated 2026-07-24", "·"},
		},
		{
			name:    "created only",
			created: &created,
			want:    []string{"created 2026-07-20"},
		},
		{
			name:    "updated only",
			updated: &updated,
			want:    []string{"updated 2026-07-24"},
		},
		{
			name:  "no dates",
			empty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := detailModel{issue: &issue.Issue{
				CreatedAt: tt.created,
				UpdatedAt: tt.updated,
			}}
			got := m.renderDates()
			if tt.empty {
				if got != "" {
					t.Fatalf("expected empty output, got %q", got)
				}
				return
			}
			for _, sub := range tt.want {
				if !strings.Contains(got, sub) {
					t.Errorf("output %q missing %q", got, sub)
				}
			}
		})
	}
}
