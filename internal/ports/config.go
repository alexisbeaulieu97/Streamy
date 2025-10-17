package ports

import (
	"context"

	pipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

// ConfigLoader loads pipeline definitions from an external source such as the
// filesystem, an embedded asset, or a remote service. Implementations must be
// deterministic, respect context cancellation, and translate infrastructure
// failures into domain-friendly error codes (see specs/009-domain-driven-refactor/errors.md).
//
// Error mapping expectations:
//   - io/fs.ErrNotExist → ErrCodeNotFound
//   - schema or YAML parsing failures → ErrCodeValidation
//   - context cancellation/deadline → ErrCodeCancelled or ErrCodeTimeout
//   - unexpected I/O issues → ErrCodeInternal with wrapped cause
//
// ConfigLoader is consumed exclusively by application-layer use cases; domain
// packages never depend on concrete infrastructure concerns.
type ConfigLoader interface {
	// Load materialises a fully validated pipeline from the provided location.
	// Implementations should:
	//   1. Respect ctx for cancellation/deadlines prior to expensive work.
	//   2. Parse the source into domain entities without mutating global state.
	//   3. Return rich domain errors containing contextual metadata (path, line).
	Load(ctx context.Context, path string) (*pipeline.Pipeline, error)

	// Validate performs a lightweight syntactic check without instantiating the
	// entire pipeline. This enables the CLI to surface errors quickly (e.g.
	// `streamy validate config.yaml`). Implementations must avoid side effects
	// and only return ErrCodeValidation, ErrCodeNotFound, ErrCodeCancelled, or
	// ErrCodeTimeout.
	Validate(ctx context.Context, path string) error
}
