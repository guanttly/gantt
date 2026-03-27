package pipeline

import (
	"context"
	"testing"

	"gantt-saas/internal/core/schedule/step"

	"go.uber.org/zap"
)

// stubStep 用于测试的桩步骤
type stubStep struct {
	name    string
	called  bool
	failErr error
}

func (s *stubStep) Name() string { return s.name }
func (s *stubStep) Execute(_ context.Context, _ *step.ScheduleState) error {
	s.called = true
	return s.failErr
}

func TestPipeline_RunAllSteps(t *testing.T) {
	s1 := &stubStep{name: "step1"}
	s2 := &stubStep{name: "step2"}
	s3 := &stubStep{name: "step3"}

	p := NewPipeline("test", zap.NewNop(), s1, s2, s3)
	state := step.NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-29", "user", nil)

	if err := p.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []*stubStep{s1, s2, s3} {
		if !s.called {
			t.Errorf("step %s was not called", s.name)
		}
	}
}

func TestPipeline_StopOnError(t *testing.T) {
	s1 := &stubStep{name: "step1"}
	s2 := &stubStep{name: "step2", failErr: context.DeadlineExceeded}
	s3 := &stubStep{name: "step3"}

	p := NewPipeline("test", zap.NewNop(), s1, s2, s3)
	state := step.NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-29", "user", nil)

	err := p.Run(context.Background(), state)
	if err == nil {
		t.Fatal("expected error")
	}
	if !s1.called {
		t.Error("step1 should have been called")
	}
	if !s2.called {
		t.Error("step2 should have been called")
	}
	if s3.called {
		t.Error("step3 should NOT have been called after step2 failed")
	}
}

func TestPipeline_OnProgressCallback(t *testing.T) {
	s1 := &stubStep{name: "StepA"}

	p := NewPipeline("test", zap.NewNop(), s1)
	state := step.NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-29", "user", nil)

	var progSteps []string
	state.OnProgress = func(stepName string, progress float64, message string) {
		progSteps = append(progSteps, stepName)
	}

	if err := p.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Pipeline.Run 在执行每个步骤前调用 OnProgress，完成时调用 "completed"
	if len(progSteps) < 2 {
		t.Fatalf("expected at least 2 progress callbacks, got %d: %v", len(progSteps), progSteps)
	}
	if progSteps[len(progSteps)-1] != "completed" {
		t.Errorf("last progress step should be 'completed', got %q", progSteps[len(progSteps)-1])
	}
}
