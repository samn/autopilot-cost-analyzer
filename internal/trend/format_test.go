package trend

import (
	"strings"
	"testing"
	"time"

	"github.com/samn/gke-cost-analyzer/internal/cost"
)

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"just now", now, "now"},
		{"5 seconds", now.Add(-5 * time.Second), "5s ago"},
		{"30 seconds", now.Add(-30 * time.Second), "30s ago"},
		{"2 minutes", now.Add(-2 * time.Minute), "2m ago"},
		{"90 minutes", now.Add(-90 * time.Minute), "1h ago"},
		{"3 hours", now.Add(-3 * time.Hour), "3h ago"},
		{"future", now.Add(5 * time.Second), "now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTimeAgo(tt.t, now)
			if got != tt.want {
				t.Errorf("FormatTimeAgo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatEvent_Aberration(t *testing.T) {
	now := time.Now()
	e := Event{
		Time:      now.Add(-2 * time.Minute),
		Key:       cost.GroupKey{Team: "platform", Workload: "web"},
		Kind:      EventAberration,
		PrevCost:  0.02,
		NewCost:   0.029,
		PctChange: 45,
	}
	got := FormatEvent(e, now)
	if !strings.Contains(got, "2m ago") {
		t.Errorf("expected '2m ago', got: %s", got)
	}
	if !strings.Contains(got, "platform/web") {
		t.Errorf("expected 'platform/web', got: %s", got)
	}
	if !strings.Contains(got, "cost +45%") {
		t.Errorf("expected 'cost +45%%', got: %s", got)
	}
	if !strings.Contains(got, "$0.0200") {
		t.Errorf("expected '$0.0200', got: %s", got)
	}
}

func TestFormatEvent_AberrationDecrease(t *testing.T) {
	now := time.Now()
	e := Event{
		Time:      now.Add(-30 * time.Second),
		Key:       cost.GroupKey{Team: "data", Workload: "pipeline"},
		Kind:      EventAberration,
		PrevCost:  1.0,
		NewCost:   0.5,
		PctChange: -50,
	}
	got := FormatEvent(e, now)
	if !strings.Contains(got, "cost -50%") {
		t.Errorf("expected 'cost -50%%', got: %s", got)
	}
}

func TestFormatEvent_Appeared(t *testing.T) {
	now := time.Now()
	e := Event{
		Time:    now.Add(-10 * time.Second),
		Key:     cost.GroupKey{Team: "ml", Workload: "training"},
		Kind:    EventAppeared,
		NewCost: 0.5,
	}
	got := FormatEvent(e, now)
	if !strings.Contains(got, "appeared") {
		t.Errorf("expected 'appeared', got: %s", got)
	}
	if !strings.Contains(got, "ml/training") {
		t.Errorf("expected 'ml/training', got: %s", got)
	}
}

func TestFormatEvent_Disappeared(t *testing.T) {
	now := time.Now()
	e := Event{
		Time:     now.Add(-5 * time.Second),
		Key:      cost.GroupKey{Team: "ml", Workload: "training"},
		Kind:     EventDisappeared,
		PrevCost: 0.5,
	}
	got := FormatEvent(e, now)
	if !strings.Contains(got, "disappeared") {
		t.Errorf("expected 'disappeared', got: %s", got)
	}
	if !strings.Contains(got, "was $0.5000") {
		t.Errorf("expected 'was $0.5000', got: %s", got)
	}
}

func TestFormatEvent_NoTeam(t *testing.T) {
	now := time.Now()
	e := Event{
		Time:    now,
		Key:     cost.GroupKey{Workload: "standalone"},
		Kind:    EventAppeared,
		NewCost: 0.1,
	}
	got := FormatEvent(e, now)
	if !strings.Contains(got, "standalone") {
		t.Errorf("expected 'standalone', got: %s", got)
	}
	// Should not have "/"
	if strings.Contains(got, "/standalone") {
		t.Errorf("should not have team prefix when team is empty: %s", got)
	}
}
