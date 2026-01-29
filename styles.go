package main

import "github.com/charmbracelet/lipgloss"

// Styles for the application UI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	queryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	queryBoxFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF6B6B")).
				Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#5A5A5A")).
				Padding(0, 1)

	tableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#7D56F4")).
				Foreground(lipgloss.Color("#FAFAFA"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#353533")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7D56F4")).
				MarginBottom(1)

	fieldLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Width(20)

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AFAFAF"))

	fieldInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AFAFAF"))

	readOnlyBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B6B")).
				Bold(true)

	editableBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#73F59F")).
				Bold(true)

	// NULL value styles
	nullValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C678DD")). // magenta/purple
			Italic(true)

	nullBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C678DD")).
			Bold(true)

	emptyStringStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#5C6370")). // dim gray
				Italic(true)

	// Type indicator styles for detail view
	numericValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D19A66")) // orange for numbers

	booleanValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#56B6C2")) // cyan for booleans

	// Table NULL cell style
	nullCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5C6370")).
			Italic(true).
			Padding(0, 1)
)
