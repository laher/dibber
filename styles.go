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

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#353533")).
			Padding(0, 1)
)

// GetStyles returns the ThemedStyles for the model
func (m Model) GetStyles() ThemedStyles {
	return NewThemedStyles(m.theme)
}
