package main

import (
	"context"
	"sync"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type cleanupFunc func(context.Context) error

type cleanupStep struct {
	name string
	fn   cleanupFunc
}

type cleanupStack struct {
	mu     sync.Mutex
	steps  []cleanupStep
	logger ports.Logger
	ran    bool
}

func newCleanupStack(logger ports.Logger) *cleanupStack {
	return &cleanupStack{logger: logger}
}

func (c *cleanupStack) Register(name string, fn cleanupFunc) {
	if fn == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ran {
		return
	}

	c.steps = append(c.steps, cleanupStep{name: name, fn: fn})
}

func (c *cleanupStack) Run(ctx context.Context) {
	c.mu.Lock()

	if c.ran {
		c.mu.Unlock()
		return
	}

	c.ran = true
	steps := make([]cleanupStep, len(c.steps))
	copy(steps, c.steps)
	c.steps = nil
	c.mu.Unlock()

	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if c.logger != nil && step.name != "" {
			c.logger.Info(ctx, "running cleanup", "cleanup", step.name)
		}

		if step.fn == nil {
			continue
		}

		if err := step.fn(ctx); err != nil && c.logger != nil {
			c.logger.Warn(ctx, "cleanup failed", "cleanup", step.name, "error", err)
		}
	}
}
