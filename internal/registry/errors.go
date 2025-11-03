package registry

import "errors"

// ErrSelfDependency indicates that a pipeline declares itself as a dependency.
var ErrSelfDependency = errors.New("self dependency")
