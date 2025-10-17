package ports

import (
	"context"
	"time"

	pipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

// RegistryStore persists named pipeline registrations so users can execute
// pipelines without remembering file paths. Implementations should be durable
// (e.g., file-backed) and safe for concurrent reads/writes. Error mapping rules:
//   - Missing registrations → ErrCodeNotFound
//   - Validation issues (e.g., duplicate IDs) → ErrCodeValidation/ErrCodeDuplicate
//   - I/O failures → ErrCodeExecution or ErrCodeInternal with wrapped cause
type RegistryStore interface {
	Store(ctx context.Context, registration *Registration) error
	Get(ctx context.Context, id string) (*Registration, error)
	List(ctx context.Context) ([]Registration, error)
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status ExecutionStatus) error
}

// Registration captures the metadata persisted for a pipeline entry.
type Registration struct {
	ID                  string
	Name                string
	ConfigPath          string
	RegisteredAt        time.Time
	LastVerifiedAt      *time.Time
	LastExecutionStatus ExecutionStatus
	Metadata            map[string]string
}

// ExecutionStatus records the last known verification or execution outcome.
type ExecutionStatus struct {
	Status    RegistryStatus
	Message   string
	Timestamp time.Time
	Duration  time.Duration
	Error     *string
}

// RegistryStatus represents coarse-grained pipeline health.
type RegistryStatus string

const (
	RegistryStatusSatisfied RegistryStatus = "satisfied"
	RegistryStatusDrifted   RegistryStatus = "drifted"
	RegistryStatusFailed    RegistryStatus = "failed"
	RegistryStatusUnknown   RegistryStatus = "unknown"
)

// ValidationService runs post-execution validation checks. Implementations
// should execute validations in parallel where safe, aggregate results into a
// VerificationSummary, and surface infrastructure failures via ErrCodeExecution.
type ValidationService interface {
	RunValidations(ctx context.Context, validations []pipeline.Validation) (pipeline.VerificationSummary, error)
}
