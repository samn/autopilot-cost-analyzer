package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/samn/gke-cost-analyzer/internal/cost"
	"github.com/samn/gke-cost-analyzer/internal/trend"
)

func makeEvent(kind trend.EventKind, team, workload string, pctChange float64, ago time.Duration) trend.Event {
	return trend.Event{
		Time:      time.Now().Add(-ago),
		Key:       cost.GroupKey{Team: team, Workload: workload},
		Kind:      kind,
		PrevCost:  1.0,
		NewCost:   1.0 + pctChange/100,
		PctChange: pctChange,
	}
}

func TestRenderEventLog_Empty(t *testing.T) {
	got := RenderEventLog(nil, time.Now(), 5)
	if !strings.Contains(got, "waiting for data") {
		t.Errorf("empty log should show waiting message, got: %s", got)
	}
}

func TestRenderEventLog_ShowsEvents(t *testing.T) {
	now := time.Now()
	events := []trend.Event{
		makeEvent(trend.EventAppeared, "platform", "web", 0, 2*time.Minute),
		makeEvent(trend.EventAberration, "platform", "web", 45, 30*time.Second),
	}
	got := RenderEventLog(events, now, 5)
	if !strings.Contains(got, "Events") {
		t.Errorf("should show header, got: %s", got)
	}
	if !strings.Contains(got, "platform/web") {
		t.Errorf("should show workload, got: %s", got)
	}
}

func TestRenderEventLog_MaxLines(t *testing.T) {
	now := time.Now()
	var events []trend.Event
	for i := 0; i < 20; i++ {
		events = append(events, makeEvent(
			trend.EventAberration, "team", "workload",
			float64(i*5), time.Duration(20-i)*time.Second,
		))
	}

	got := RenderEventLog(events, now, 5)
	// Should have header + 5 event lines = 6 lines total.
	lines := strings.Split(got, "\n")
	if len(lines) != 6 {
		t.Errorf("expected 6 lines (header + 5 events), got %d", len(lines))
	}
}

func TestRenderEventLog_ShowsMostRecent(t *testing.T) {
	now := time.Now()
	events := []trend.Event{
		makeEvent(trend.EventAberration, "team", "old", 10, 5*time.Minute),
		makeEvent(trend.EventAberration, "team", "new", 20, 5*time.Second),
	}

	got := RenderEventLog(events, now, 1)
	// With maxLines=1, should show only the most recent event.
	if !strings.Contains(got, "team/new") {
		t.Errorf("should show most recent event, got: %s", got)
	}
}
