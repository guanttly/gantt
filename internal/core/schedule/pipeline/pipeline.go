package pipeline

import (
"context"
"fmt"

"gantt-saas/internal/core/schedule/step"

"go.uber.org/zap"
)

// Pipeline 排班管道执行器。
type Pipeline struct {
name   string
steps  []step.Step
logger *zap.Logger
}

// NewPipeline 创建排班管道。
func NewPipeline(name string, logger *zap.Logger, steps ...step.Step) *Pipeline {
return &Pipeline{
name:   name,
steps:  steps,
logger: logger,
}
}

// Run 执行管道中的所有步骤。
func (p *Pipeline) Run(ctx context.Context, state *step.ScheduleState) error {
p.logger.Info("pipeline started", zap.String("pipeline", p.name))

for i, s := range p.steps {
p.logger.Info("step started",
zap.String("pipeline", p.name),
zap.String("step", s.Name()),
zap.Int("index", i+1),
zap.Int("total", len(p.steps)),
)

if state.OnProgress != nil {
progress := float64(i) / float64(len(p.steps))
state.OnProgress(s.Name(), progress, "执行中")
}

if err := s.Execute(ctx, state); err != nil {
p.logger.Error("step failed",
zap.String("pipeline", p.name),
zap.String("step", s.Name()),
zap.Error(err),
)
return fmt.Errorf("step '%s' failed: %w", s.Name(), err)
}

p.logger.Info("step completed",
zap.String("pipeline", p.name),
zap.String("step", s.Name()),
)
}

if state.OnProgress != nil {
state.OnProgress("completed", 1.0, "排班完成")
}

p.logger.Info("pipeline completed",
zap.String("pipeline", p.name),
zap.Int("assignments", len(state.Assignments)),
zap.Int("violations", len(state.Violations)),
)
return nil
}

// Name 返回管道名称。
func (p *Pipeline) Name() string {
return p.name
}
