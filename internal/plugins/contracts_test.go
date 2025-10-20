package plugins_test

import (
	"context"
	"testing"

	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	commandplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	copyplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	lineinfileplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"
	packageplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	repoplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	symlinkplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
	templateplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/template"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type pluginContractCase struct {
	name string
	typ  domainplugin.Type
	ctor func() ports.Plugin
}

func TestPortsPluginContracts(t *testing.T) {
	cases := []pluginContractCase{
		{name: "command", typ: domainplugin.TypeCommand, ctor: commandplugin.New},
		{name: "copy", typ: domainplugin.TypeCopy, ctor: copyplugin.New},
		{name: "line_in_file", typ: domainplugin.TypeLineInFile, ctor: lineinfileplugin.New},
		{name: "package", typ: domainplugin.TypePackage, ctor: packageplugin.New},
		{name: "repo", typ: domainplugin.TypeRepo, ctor: repoplugin.New},
		{name: "symlink", typ: domainplugin.TypeSymlink, ctor: symlinkplugin.New},
		{name: "template", typ: domainplugin.TypeTemplate, ctor: templateplugin.New},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := tc.ctor()

			meta := plugin.Metadata()
			require.NoError(t, meta.Validate(), "metadata must be valid")
			require.Equal(t, tc.typ, meta.Type, "metadata type mismatch")

			step := domainpipeline.Step{
				ID:     "contract_" + tc.name,
				Type:   domainpipeline.StepType(tc.typ),
				Config: map[string]interface{}{"contract": "value"},
			}

			cancelledCtx, cancel := context.WithCancel(context.Background())
			cancel()

			_, evalErr := plugin.Evaluate(cancelledCtx, step)
			require.Error(t, evalErr, "canceled evaluate should error")

			var evalDomainErr *domainpipeline.DomainError
			require.ErrorAs(t, evalErr, &evalDomainErr)
			require.Equal(t, domainpipeline.ErrCodeCancelled, evalDomainErr.Code, "evaluate should surface cancellation error")

			_, applyErr := plugin.Apply(cancelledCtx, nil, step)
			require.Error(t, applyErr, "canceled apply should error")

			var applyDomainErr *domainpipeline.DomainError
			require.ErrorAs(t, applyErr, &applyDomainErr)
			require.Equal(t, domainpipeline.ErrCodeCancelled, applyDomainErr.Code, "apply should surface cancellation error")
		})
	}
}
