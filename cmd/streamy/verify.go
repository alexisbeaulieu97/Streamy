package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type verifyOptions struct {
	ConfigPath string
	Verbose    bool
	JSON       bool
	Timeout    time.Duration
}

var verifyCmdRunner = runVerify

func newVerifyCmd(root *rootFlags) *cobra.Command {
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

			return verifyCmdRunner(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output results in JSON format")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 30*time.Second, "Default timeout per step; accepts Go duration strings (e.g. 60s)")

	return cmd
}

func runVerify(opts verifyOptions) error {
	// Parse configuration
	cfg, err := config.ParseConfig(opts.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing configuration: %v\n", err)
		os.Exit(2)
	}

	// Create logger
	level := "info"
	if opts.Verbose {
		level = "debug"
	}

	log, err := logger.New(logger.Options{Level: level, HumanReadable: !opts.JSON})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(3)
	}

	// Create executor
	executor := engine.NewExecutor(log)

	// Run verification
	ctx := context.Background()
	perStepTimeout := opts.Timeout
	if perStepTimeout > 0 {
		var cancel context.CancelFunc
		totalTimeout := perStepTimeout * time.Duration(len(cfg.Steps))
		if len(cfg.Steps) == 0 {
			totalTimeout = perStepTimeout
		}
		ctx, cancel = context.WithTimeout(ctx, totalTimeout)
		defer cancel()
	}

	log.WithFields(map[string]interface{}{
		"config": opts.ConfigPath,
		"steps":  len(cfg.Steps),
	}).Info("Starting verification")

	summary, err := executor.VerifySteps(ctx, cfg.Steps, perStepTimeout)
	if err != nil {
		var validationErr *streamyerrors.ValidationError
		if errors.As(err, &validationErr) {
			fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
			os.Exit(2)
		}

		fmt.Fprintf(os.Stderr, "Verification error: %v\n", err)
		os.Exit(3)
	}

	log.WithFields(map[string]interface{}{
		"total":     summary.TotalSteps,
		"satisfied": summary.Satisfied,
		"missing":   summary.Missing,
		"drifted":   summary.Drifted,
		"blocked":   summary.Blocked,
		"unknown":   summary.Unknown,
		"duration":  summary.Duration.String(),
	}).Info("Verification complete")

	// Output results
	if opts.JSON {
		printJSONOutput(summary, opts.ConfigPath)
	} else if opts.Verbose {
		printVerboseOutput(summary)
	} else {
		printTableOutput(summary)
	}

	// Exit with appropriate code
	os.Exit(summary.ExitCode())
	return nil
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

func printJSONOutput(summary *model.VerificationSummary, configPath string) {
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
	encoder.Encode(jsonOutput)
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
