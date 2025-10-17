package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/pipelineconv"
)

type verifyOptions struct {
	ConfigPath string
	Verbose    bool
	JSON       bool
	Timeout    time.Duration
}

var (
	exitFunc                         = os.Exit
	stderrWriter           io.Writer = os.Stderr
	printTableOutputFunc             = printTableOutput
	printVerboseOutputFunc           = printVerboseOutput
	printJSONOutputFunc              = printJSONOutput
)

func newVerifyCmd(root *rootFlags, app *AppContext) *cobra.Command {
	opts := verifyOptions{}

	cmd := &cobra.Command{
		Use:   "verify <config-file>",
		Short: "Verify system state matches configuration without making changes",
		Long: `Verify performs read-only checks to determine if the system state matches
the declared configuration. Returns exit code 0 if all steps are satisfied,
exit code 1 if any changes are needed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ConfigPath = args[0]
			opts.Verbose = root.verbose

			return runVerify(cmd.Context(), app, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output results in JSON format")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 30*time.Second, "Default timeout per step; accepts Go duration strings (e.g. 60s)")

	return cmd
}

func runVerify(ctx context.Context, app *AppContext, opts verifyOptions) error {
	exitCode, err := runVerifyInternal(ctx, app, opts)
	if err != nil {
		return err
	}
	exitFunc(exitCode)
	return nil
}

func runVerifyInternal(ctx context.Context, app *AppContext, opts verifyOptions) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	preparedPipeline, _, err := app.PrepareUseCase.Prepare(ctx, opts.ConfigPath)
	if err != nil {
		return handleVerifyPrepareError(err)
	}

	execCtx := ctx
	if opts.Timeout > 0 {
		stepCount := len(preparedPipeline.Steps)
		totalTimeout := opts.Timeout * time.Duration(stepCount)
		if stepCount == 0 {
			totalTimeout = opts.Timeout
		}
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, totalTimeout)
		defer cancel()
	}

	verifiedPipeline, results, verifyErr := app.VerifyUseCase.Verify(execCtx, opts.ConfigPath)
	if verifyErr != nil {
		return handleVerifyExecutionError(verifyErr)
	}

	if verifiedPipeline == nil {
		verifiedPipeline = preparedPipeline
	}

	summary := pipelineconv.BuildVerificationSummary(verifiedPipeline, results)

	if opts.JSON {
		if err := printJSONOutputFunc(summary, opts.ConfigPath); err != nil {
			_, _ = fmt.Fprintf(stderrWriter, "Failed to generate JSON output: %v\n", err)
			return 3, nil
		}
	} else if opts.Verbose {
		printVerboseOutputFunc(summary)
	} else {
		printTableOutputFunc(summary)
	}

	return summary.ExitCode(), nil
}

func handleVerifyPrepareError(err error) (int, error) {
	switch {
	case pipelineconv.IsParseError(err):
		_, _ = fmt.Fprintf(stderrWriter, "Error parsing configuration: %v\n", err)
		return 2, nil
	case pipelineconv.IsConfigError(err):
		_, _ = fmt.Fprintf(stderrWriter, "Configuration error: %v\n", err)
		return 2, nil
	default:
		return 3, err
	}
}

func handleVerifyExecutionError(err error) (int, error) {
	if pipelineconv.IsConfigError(err) {
		_, _ = fmt.Fprintf(stderrWriter, "Configuration error: %v\n", err)
		return 2, nil
	}
	_, _ = fmt.Fprintf(stderrWriter, "Verification error: %v\n", err)
	return 3, nil
}

func printTableOutput(summary *model.VerificationSummary) {
	// Print header
	fmt.Println("\nVerification Results:")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-40s %-12s %-8s %s\n", "Step ID", "Status", "Duration", "Message")
	fmt.Println(strings.Repeat("-", 80))

	// Print each result
	for _, result := range summary.Results {
		symbol := getStatusSymbol(result.Status)
		duration := fmt.Sprintf("%.2fs", result.Duration.Seconds())
		message := truncateString(result.Message, 40)

		fmt.Printf("%-40s %-12s %-8s %s\n",
			truncateString(result.StepID, 40),
			fmt.Sprintf("%s %s", symbol, result.Status),
			duration,
			message,
		)
	}

	// Print summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total:     %d\n", summary.TotalSteps)
	fmt.Printf("  ✔ Satisfied: %d\n", summary.Satisfied)
	fmt.Printf("  ✖ Missing:   %d\n", summary.Missing)
	fmt.Printf("  ⚠ Drifted:   %d\n", summary.Drifted)
	fmt.Printf("  🚫 Blocked:  %d\n", summary.Blocked)
	fmt.Printf("  ? Unknown:  %d\n", summary.Unknown)
	fmt.Printf("  Duration:  %s\n", summary.Duration.String())

	if summary.AllSatisfied() {
		fmt.Println("\n✅ All steps satisfied - no changes needed")
	} else {
		fmt.Println("\n❌ Changes needed - run 'streamy apply' to fix")
	}
}

func printVerboseOutput(summary *model.VerificationSummary) {
	printTableOutput(summary)

	// Print detailed information for drifted/blocked steps
	hasDetails := false
	for _, result := range summary.Results {
		if result.Status == model.StatusDrifted && result.Details != "" {
			if !hasDetails {
				fmt.Println("\nDetailed Diff Output:")
				fmt.Println(strings.Repeat("=", 80))
				hasDetails = true
			}
			fmt.Printf("\n--- Step: %s ---\n", result.StepID)
			fmt.Println(result.Details)
		}
		if result.Status == model.StatusBlocked && result.Error != nil {
			if !hasDetails {
				fmt.Println("\nError Details:")
				fmt.Println(strings.Repeat("=", 80))
				hasDetails = true
			}
			fmt.Printf("\n--- Step: %s ---\n", result.StepID)
			fmt.Printf("Error: %v\n", result.Error)
		}
	}
}

func printJSONOutput(summary *model.VerificationSummary, configPath string) error {
	// Convert to JSON-friendly format
	type JSONResult struct {
		StepID    string  `json:"step_id"`
		Status    string  `json:"status"`
		Message   string  `json:"message"`
		Details   string  `json:"details,omitempty"`
		Error     string  `json:"error,omitempty"`
		Duration  float64 `json:"duration_seconds"`
		Timestamp string  `json:"timestamp"`
	}

	type JSONSummary struct {
		TotalSteps int     `json:"total_steps"`
		Satisfied  int     `json:"satisfied"`
		Missing    int     `json:"missing"`
		Drifted    int     `json:"drifted"`
		Blocked    int     `json:"blocked"`
		Unknown    int     `json:"unknown"`
		Duration   float64 `json:"duration_seconds"`
	}

	type JSONOutput struct {
		ConfigFile string       `json:"config_file"`
		Summary    JSONSummary  `json:"summary"`
		Results    []JSONResult `json:"results"`
	}

	jsonOutput := JSONOutput{
		ConfigFile: configPath,
		Summary: JSONSummary{
			TotalSteps: summary.TotalSteps,
			Satisfied:  summary.Satisfied,
			Missing:    summary.Missing,
			Drifted:    summary.Drifted,
			Blocked:    summary.Blocked,
			Unknown:    summary.Unknown,
			Duration:   summary.Duration.Seconds(),
		},
		Results: make([]JSONResult, len(summary.Results)),
	}

	for i, result := range summary.Results {
		jsonResult := JSONResult{
			StepID:    result.StepID,
			Status:    string(result.Status),
			Message:   result.Message,
			Details:   result.Details,
			Duration:  result.Duration.Seconds(),
			Timestamp: result.Timestamp.Format(time.RFC3339),
		}
		if result.Error != nil {
			jsonResult.Error = result.Error.Error()
		}
		jsonOutput.Results[i] = jsonResult
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(jsonOutput); err != nil {
		return err
	}
	return nil
}

func getStatusSymbol(status model.VerificationStatus) string {
	switch status {
	case model.StatusSatisfied:
		return "✔"
	case model.StatusMissing:
		return "✖"
	case model.StatusDrifted:
		return "⚠"
	case model.StatusBlocked:
		return "🚫"
	case model.StatusUnknown:
		return "?"
	default:
		return "?"
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
