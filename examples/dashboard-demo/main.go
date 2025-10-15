package main

import (
	"fmt"

	"github.com/alexisbeaulieu97/streamy/internal/ui/components"
)

func main() {
	// Simulate a CLI dashboard view
	dashboard := buildDashboard()
	fmt.Println(dashboard.View())
}

func buildDashboard() *components.Card {
	// Header section with title and status badge
	header := components.HStack(
		components.NewHeader("Streamy Pipeline Dashboard").
			WithAppliers(components.Typography(components.TypographyVariantTitle)),
		components.SuccessBadge("Active"),
	).WithGap(2).WithCrossAlign(components.CrossCenter)

	// System stats panel
	statsPanel := components.NewPanel(
		components.VStack(
			buildStatRow("CPU Usage", "45%", components.SuccessBadge),
			buildStatRow("Memory", "78%", components.WarningBadge),
			buildStatRow("Disk I/O", "32%", components.SuccessBadge),
			buildStatRow("Network", "12%", components.SuccessBadge),
		).WithGap(1),
	).WithTitle("System Resources")

	// Pipeline status section
	pipelineStatus := components.NewPanel(
		components.VStack(
			buildPipelineItem("dev-environment", "satisfied", "2m ago"),
			buildPipelineItem("prod-setup", "drifted", "1h ago"),
			buildPipelineItem("test-config", "satisfied", "5m ago"),
			buildPipelineItem("backup-job", "failed", "just now"),
		).WithGap(1),
	).WithTitle("Pipeline Status")

	// Recent activity
	activityPanel := components.NewPanel(
		components.VStack(
			components.InfoAlert("Pipeline 'dev-environment' completed successfully"),
			components.WarningAlert("Pipeline 'prod-setup' detected drift in 3 resources"),
			components.ErrorAlert("Pipeline 'backup-job' failed: connection timeout"),
		).WithGap(1),
	).WithTitle("Recent Activity")

	// Action buttons
	actionBar := components.HStack(
		components.PrimaryButton("Refresh All"),
		components.SecondaryButton("Verify"),
		components.InfoButton("Settings"),
	).WithGap(2)

	// Assemble the dashboard
	return components.NewCard(
		components.VStack(
			header,
			components.HorizontalDivider(),
			statsPanel,
			components.VerticalSpacer(1),
			pipelineStatus,
			components.VerticalSpacer(1),
			activityPanel,
			components.HorizontalDivider(),
			actionBar,
		).WithGap(1),
	)
}

func buildStatRow(label, value string, badgeFunc func(string) *components.Badge) *components.Stack {
	return components.HStack(
		components.EmphasisText(label + ":"),
		badgeFunc(value),
	).WithGap(1)
}

func buildPipelineItem(name, status, time string) *components.Stack {
	var statusBadge *components.Badge
	var icon string

	switch status {
	case "satisfied":
		statusBadge = components.SuccessBadge("satisfied")
		icon = "✓"
	case "drifted":
		statusBadge = components.WarningBadge("drifted")
		icon = "⚠"
	case "failed":
		statusBadge = components.ErrorBadge("failed")
		icon = "✗"
	default:
		statusBadge = components.NewBadge("unknown")
		icon = "?"
	}

	nameText := components.EmphasisText(name)
	timeText := components.NewText(time).WithAppliers(components.Typography(components.TypographyVariantTextSm))
	
	return components.HStack(
		components.NewText(icon),
		nameText,
		statusBadge,
		components.HorizontalSpacer(2),
		timeText,
	).WithGap(1)
}
