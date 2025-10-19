package testutil

import (
	"context"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// MockValidationService implements ports.ValidationService for tests.
type MockValidationService struct {
	Calls   []context.Context
	RunFunc func(ctx context.Context, validations []domainpipeline.Validation) (domainpipeline.VerificationSummary, error)
}

func (m *MockValidationService) RunValidations(ctx context.Context, validations []domainpipeline.Validation) (domainpipeline.VerificationSummary, error) {
	if m == nil {
		return domainpipeline.VerificationSummary{}, nil
	}
	m.Calls = append(m.Calls, ctx)
	if m.RunFunc != nil {
		return m.RunFunc(ctx, validations)
	}
	return domainpipeline.VerificationSummary{}, nil
}

var _ ports.ValidationService = (*MockValidationService)(nil)
