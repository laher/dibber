package main

import "github.com/charmbracelet/lipgloss"

// ThemedStyles holds all the lipgloss styles for a given theme
type ThemedStyles struct {
	Title           lipgloss.Style
	QueryBox        lipgloss.Style
	QueryBoxFocused lipgloss.Style
	TableHeader     lipgloss.Style
	TableCell       lipgloss.Style
	SelectedRow     lipgloss.Style
	StatusBar       lipgloss.Style
	Error           lipgloss.Style
	Help            lipgloss.Style
	DetailTitle     lipgloss.Style
	FieldLabel      lipgloss.Style
	FieldValue      lipgloss.Style
	FieldInput      lipgloss.Style
	ReadOnlyBadge   lipgloss.Style
	EditableBadge   lipgloss.Style
	NullValue       lipgloss.Style
	NullBadge       lipgloss.Style
	EmptyString     lipgloss.Style
	NumericValue    lipgloss.Style
	BooleanValue    lipgloss.Style
	NullCell        lipgloss.Style
}

// NewThemedStyles creates a new ThemedStyles from a Theme
func NewThemedStyles(t Theme) ThemedStyles {
	return ThemedStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.TextBright).
			Background(t.Primary).
			Padding(0, 1),

		QueryBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(0, 1),

		QueryBoxFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Success).
			Padding(0, 1),

		TableHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.TextBright).
			Background(t.Secondary).
			Padding(0, 1),

		TableCell: lipgloss.NewStyle().
			Padding(0, 1),

		SelectedRow: lipgloss.NewStyle().
			Background(t.Primary).
			Foreground(t.TextBright),

		StatusBar: lipgloss.NewStyle().
			Foreground(t.TextBright).
			Background(t.Secondary).
			Padding(0, 1),

		Error: lipgloss.NewStyle().
			Foreground(t.Danger).
			Bold(true),

		Help: lipgloss.NewStyle().
			Foreground(t.TextDim),

		DetailTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary).
			MarginBottom(1),

		FieldLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.TextBright).
			Width(20),

		FieldValue: lipgloss.NewStyle().
			Foreground(t.TextNormal),

		FieldInput: lipgloss.NewStyle().
			Foreground(t.TextNormal),

		ReadOnlyBadge: lipgloss.NewStyle().
			Foreground(t.Danger).
			Bold(true),

		EditableBadge: lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true),

		NullValue: lipgloss.NewStyle().
			Foreground(t.SyntaxNull).
			Italic(true),

		NullBadge: lipgloss.NewStyle().
			Foreground(t.SyntaxNull).
			Bold(true),

		EmptyString: lipgloss.NewStyle().
			Foreground(t.TextDim).
			Italic(true),

		NumericValue: lipgloss.NewStyle().
			Foreground(t.SyntaxNumber),

		BooleanValue: lipgloss.NewStyle().
			Foreground(t.SyntaxBoolean),

		NullCell: lipgloss.NewStyle().
			Foreground(t.TextDim).
			Italic(true).
			Padding(0, 1),
	}
}

// Default styles (for backwards compatibility and non-themed contexts)
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

// GetStyles returns the ThemedStyles for the model
func (m Model) GetStyles() ThemedStyles {
	return NewThemedStyles(m.theme)
}
