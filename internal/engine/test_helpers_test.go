package engine

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
)

func stepWithConfig(t *testing.T, base config.Step, cfg any) config.Step {
	t.Helper()
	require.NoError(t, base.SetConfig(cfg))
	return base
}
