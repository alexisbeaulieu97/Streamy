package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type removeOptions struct {
	force bool
}

func newRemoveCmd(rootFlags *rootFlags) *cobra.Command {
	opts := &removeOptions{}

	cmd := &cobra.Command{
		Use:   "remove <pipeline-id>",
		Short: "Remove a pipeline from the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd, args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Remove without confirmation")

	return cmd
}

func runRemove(cmd *cobra.Command, pipelineID string, opts *removeOptions) error {
	if strings.TrimSpace(pipelineID) == "" {
		return newCommandError("remove", "validating pipeline ID", errors.New("pipeline ID cannot be empty"), "Provide the pipeline ID you wish to remove.")
	}

	registryPath, err := defaultRegistryPath()
	if err != nil {
		return newCommandError("remove", "determining registry path", err, "Ensure your HOME directory is set correctly.")
	}

	statusPath, err := defaultStatusCachePath()
	if err != nil {
		return newCommandError("remove", "determining status cache path", err, "Ensure your HOME directory is set correctly.")
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		return newCommandError("remove", "loading registry", err, "Check registry file permissions and try again.")
	}

	pipeline, err := reg.Get(pipelineID)
	if err != nil {
		return newCommandError("remove", fmt.Sprintf("looking up pipeline %q", pipelineID), err, "Run 'streamy registry list' to view registered pipelines.")
	}

	if !opts.force {
		confirmed, err := confirmRemoval(cmd, pipelineID, pipeline.Name)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}
	}

	if err := reg.Remove(pipelineID); err != nil {
		return newCommandError("remove", fmt.Sprintf("removing pipeline %q", pipelineID), err, "Verify the pipeline still exists using 'streamy registry list'.")
	}

	if err := reg.Save(); err != nil {
		return newCommandError("remove", "saving registry", err, "Check disk space and file permissions, then retry.")
	}

	statusCache, err := registry.NewStatusCache(statusPath)
	if err == nil {
		_ = statusCache.Invalidate(pipelineID)
		_ = statusCache.Save()
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed pipeline '%s'\n", pipelineID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nThe configuration file at %s was not deleted.\n", pipeline.Path)

	return nil
}

func confirmRemoval(cmd *cobra.Command, pipelineID, pipelineName string) (bool, error) {
	if !isTerminal(cmd.InOrStdin()) {
		return false, newCommandError("remove", "prompting for confirmation", errors.New("not a terminal"), "Use --force when running in non-interactive environments.")
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Remove pipeline '%s' (%s) from registry? [y/N]: ", pipelineID, pipelineName)

	scanner := bufio.NewScanner(cmd.InOrStdin())
	if !scanner.Scan() {
		return false, scanner.Err()
	}

	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

func isTerminal(reader any) bool {
	if file, ok := reader.(*os.File); ok {
		return termIsTerminal(int(file.Fd()))
	}
	return false
}

var termIsTerminal = func(fd int) bool {
	return term.IsTerminal(fd)
}
