package main

import "github.com/charmbracelet/lipgloss"

// Theme defines the colors for the UI
type Theme struct {
	Name        string
	Description string

	// Primary accent color (title bar, selected rows, focused elements)
	Primary lipgloss.Color
	// Secondary accent (unfocused borders, subtle highlights)
	Secondary lipgloss.Color
	// Danger color (errors, destructive actions)
	Danger lipgloss.Color
	// Success color (editable badge, positive states)
	Success lipgloss.Color
	// Warning color (production warning, caution)
	Warning lipgloss.Color

	// Text colors
	TextBright lipgloss.Color // bright text on colored backgrounds
	TextNormal lipgloss.Color // normal text
	TextDim    lipgloss.Color // dimmed text, help text

	// Syntax highlighting
	SyntaxString   lipgloss.Color // strings, text values
	SyntaxNumber   lipgloss.Color // numeric values
	SyntaxKeyword  lipgloss.Color // SQL keywords
	SyntaxNull     lipgloss.Color // NULL values
	SyntaxBoolean  lipgloss.Color // boolean values
	SyntaxDatetime lipgloss.Color // datetime values
	SyntaxFunction lipgloss.Color // function names (COUNT, SUM, etc)
	SyntaxComment  lipgloss.Color // SQL comments
	SyntaxOperator lipgloss.Color // operators (=, <>, +, etc)
}

// Available themes
var Themes = map[string]Theme{
	"default": {
		Name:           "default",
		Description:    "Default purple theme",
		Primary:        lipgloss.Color("#7D56F4"),
		Secondary:      lipgloss.Color("#5A5A5A"),
		Danger:         lipgloss.Color("#FF6B6B"),
		Success:        lipgloss.Color("#73F59F"),
		Warning:        lipgloss.Color("#FFCC00"),
		TextBright:     lipgloss.Color("#FAFAFA"),
		TextNormal:     lipgloss.Color("#AFAFAF"),
		TextDim:        lipgloss.Color("#626262"),
		SyntaxString:   lipgloss.Color("#98C379"),
		SyntaxNumber:   lipgloss.Color("#D19A66"),
		SyntaxKeyword:  lipgloss.Color("#C678DD"),
		SyntaxNull:     lipgloss.Color("#C678DD"),
		SyntaxBoolean:  lipgloss.Color("#56B6C2"),
		SyntaxDatetime: lipgloss.Color("#E5C07B"),
		SyntaxFunction: lipgloss.Color("#61AFEF"),
		SyntaxComment:  lipgloss.Color("#5C6370"),
		SyntaxOperator: lipgloss.Color("#ABB2BF"),
	},

	"dracula": {
		Name:           "dracula",
		Description:    "Dracula dark theme",
		Primary:        lipgloss.Color("#BD93F9"), // purple
		Secondary:      lipgloss.Color("#44475A"), // current line
		Danger:         lipgloss.Color("#FF5555"), // red
		Success:        lipgloss.Color("#50FA7B"), // green
		Warning:        lipgloss.Color("#FFB86C"), // orange
		TextBright:     lipgloss.Color("#F8F8F2"), // foreground
		TextNormal:     lipgloss.Color("#BFBFBF"),
		TextDim:        lipgloss.Color("#6272A4"), // comment
		SyntaxString:   lipgloss.Color("#F1FA8C"), // yellow
		SyntaxNumber:   lipgloss.Color("#BD93F9"), // purple
		SyntaxKeyword:  lipgloss.Color("#FF79C6"), // pink
		SyntaxNull:     lipgloss.Color("#6272A4"), // comment
		SyntaxBoolean:  lipgloss.Color("#8BE9FD"), // cyan
		SyntaxDatetime: lipgloss.Color("#FFB86C"), // orange
		SyntaxFunction: lipgloss.Color("#50FA7B"), // green
		SyntaxComment:  lipgloss.Color("#6272A4"), // comment
		SyntaxOperator: lipgloss.Color("#FF79C6"), // pink
	},

	"monokai": {
		Name:           "monokai",
		Description:    "Classic Monokai theme",
		Primary:        lipgloss.Color("#A6E22E"), // green
		Secondary:      lipgloss.Color("#49483E"), // line highlight
		Danger:         lipgloss.Color("#F92672"), // pink/red
		Success:        lipgloss.Color("#A6E22E"), // green
		Warning:        lipgloss.Color("#E6DB74"), // yellow
		TextBright:     lipgloss.Color("#F8F8F2"), // foreground
		TextNormal:     lipgloss.Color("#CFCFC2"),
		TextDim:        lipgloss.Color("#75715E"), // comment
		SyntaxString:   lipgloss.Color("#E6DB74"), // yellow
		SyntaxNumber:   lipgloss.Color("#AE81FF"), // purple
		SyntaxKeyword:  lipgloss.Color("#F92672"), // pink
		SyntaxNull:     lipgloss.Color("#AE81FF"), // purple
		SyntaxBoolean:  lipgloss.Color("#66D9EF"), // cyan
		SyntaxDatetime: lipgloss.Color("#FD971F"), // orange
		SyntaxFunction: lipgloss.Color("#66D9EF"), // cyan
		SyntaxComment:  lipgloss.Color("#75715E"), // comment
		SyntaxOperator: lipgloss.Color("#F92672"), // pink
	},

	"nord": {
		Name:           "nord",
		Description:    "Arctic Nord theme",
		Primary:        lipgloss.Color("#88C0D0"), // frost cyan
		Secondary:      lipgloss.Color("#3B4252"), // polar night
		Danger:         lipgloss.Color("#BF616A"), // aurora red
		Success:        lipgloss.Color("#A3BE8C"), // aurora green
		Warning:        lipgloss.Color("#EBCB8B"), // aurora yellow
		TextBright:     lipgloss.Color("#ECEFF4"), // snow storm
		TextNormal:     lipgloss.Color("#D8DEE9"),
		TextDim:        lipgloss.Color("#4C566A"), // polar night light
		SyntaxString:   lipgloss.Color("#A3BE8C"), // green
		SyntaxNumber:   lipgloss.Color("#B48EAD"), // purple
		SyntaxKeyword:  lipgloss.Color("#81A1C1"), // frost blue
		SyntaxNull:     lipgloss.Color("#4C566A"), // dim
		SyntaxBoolean:  lipgloss.Color("#88C0D0"), // cyan
		SyntaxDatetime: lipgloss.Color("#EBCB8B"), // yellow
		SyntaxFunction: lipgloss.Color("#88C0D0"), // frost cyan
		SyntaxComment:  lipgloss.Color("#4C566A"), // polar night light
		SyntaxOperator: lipgloss.Color("#81A1C1"), // frost blue
	},

	"gruvbox": {
		Name:           "gruvbox",
		Description:    "Retro Gruvbox theme",
		Primary:        lipgloss.Color("#FE8019"), // orange
		Secondary:      lipgloss.Color("#3C3836"), // bg1
		Danger:         lipgloss.Color("#FB4934"), // red
		Success:        lipgloss.Color("#B8BB26"), // green
		Warning:        lipgloss.Color("#FABD2F"), // yellow
		TextBright:     lipgloss.Color("#EBDBB2"), // fg
		TextNormal:     lipgloss.Color("#D5C4A1"), // fg2
		TextDim:        lipgloss.Color("#928374"), // gray
		SyntaxString:   lipgloss.Color("#B8BB26"), // green
		SyntaxNumber:   lipgloss.Color("#D3869B"), // purple
		SyntaxKeyword:  lipgloss.Color("#FB4934"), // red
		SyntaxNull:     lipgloss.Color("#928374"), // gray
		SyntaxBoolean:  lipgloss.Color("#8EC07C"), // aqua
		SyntaxDatetime: lipgloss.Color("#FABD2F"), // yellow
		SyntaxFunction: lipgloss.Color("#8EC07C"), // aqua
		SyntaxComment:  lipgloss.Color("#928374"), // gray
		SyntaxOperator: lipgloss.Color("#FE8019"), // orange
	},

	"tokyo-night": {
		Name:           "tokyo-night",
		Description:    "Tokyo Night theme",
		Primary:        lipgloss.Color("#7AA2F7"), // blue
		Secondary:      lipgloss.Color("#1A1B26"), // bg dark
		Danger:         lipgloss.Color("#F7768E"), // red
		Success:        lipgloss.Color("#9ECE6A"), // green
		Warning:        lipgloss.Color("#E0AF68"), // yellow
		TextBright:     lipgloss.Color("#C0CAF5"), // foreground
		TextNormal:     lipgloss.Color("#A9B1D6"),
		TextDim:        lipgloss.Color("#565F89"), // comment
		SyntaxString:   lipgloss.Color("#9ECE6A"), // green
		SyntaxNumber:   lipgloss.Color("#FF9E64"), // orange
		SyntaxKeyword:  lipgloss.Color("#BB9AF7"), // purple
		SyntaxNull:     lipgloss.Color("#565F89"), // comment
		SyntaxBoolean:  lipgloss.Color("#7DCFFF"), // cyan
		SyntaxDatetime: lipgloss.Color("#E0AF68"), // yellow
		SyntaxFunction: lipgloss.Color("#7AA2F7"), // blue
		SyntaxComment:  lipgloss.Color("#565F89"), // comment
		SyntaxOperator: lipgloss.Color("#89DDFF"), // cyan
	},

	"catppuccin": {
		Name:           "catppuccin",
		Description:    "Catppuccin Mocha theme",
		Primary:        lipgloss.Color("#CBA6F7"), // mauve
		Secondary:      lipgloss.Color("#313244"), // surface0
		Danger:         lipgloss.Color("#F38BA8"), // red
		Success:        lipgloss.Color("#A6E3A1"), // green
		Warning:        lipgloss.Color("#F9E2AF"), // yellow
		TextBright:     lipgloss.Color("#CDD6F4"), // text
		TextNormal:     lipgloss.Color("#BAC2DE"), // subtext1
		TextDim:        lipgloss.Color("#6C7086"), // overlay0
		SyntaxString:   lipgloss.Color("#A6E3A1"), // green
		SyntaxNumber:   lipgloss.Color("#FAB387"), // peach
		SyntaxKeyword:  lipgloss.Color("#CBA6F7"), // mauve
		SyntaxNull:     lipgloss.Color("#6C7086"), // overlay0
		SyntaxBoolean:  lipgloss.Color("#89DCEB"), // sky
		SyntaxDatetime: lipgloss.Color("#F9E2AF"), // yellow
		SyntaxFunction: lipgloss.Color("#89B4FA"), // blue
		SyntaxComment:  lipgloss.Color("#6C7086"), // overlay0
		SyntaxOperator: lipgloss.Color("#89DCEB"), // sky
	},

	"solarized": {
		Name:           "solarized",
		Description:    "Solarized Dark theme",
		Primary:        lipgloss.Color("#268BD2"), // blue
		Secondary:      lipgloss.Color("#073642"), // base02
		Danger:         lipgloss.Color("#DC322F"), // red
		Success:        lipgloss.Color("#859900"), // green
		Warning:        lipgloss.Color("#B58900"), // yellow
		TextBright:     lipgloss.Color("#FDF6E3"), // base3
		TextNormal:     lipgloss.Color("#839496"), // base0
		TextDim:        lipgloss.Color("#586E75"), // base01
		SyntaxString:   lipgloss.Color("#2AA198"), // cyan
		SyntaxNumber:   lipgloss.Color("#D33682"), // magenta
		SyntaxKeyword:  lipgloss.Color("#859900"), // green
		SyntaxNull:     lipgloss.Color("#586E75"), // base01
		SyntaxBoolean:  lipgloss.Color("#2AA198"), // cyan
		SyntaxDatetime: lipgloss.Color("#B58900"), // yellow
		SyntaxFunction: lipgloss.Color("#268BD2"), // blue
		SyntaxComment:  lipgloss.Color("#586E75"), // base01
		SyntaxOperator: lipgloss.Color("#839496"), // base0
	},

	// Special "warning" themes for production/sensitive environments
	"production": {
		Name:           "production",
		Description:    "Red warning theme for production databases",
		Primary:        lipgloss.Color("#FF4444"), // red
		Secondary:      lipgloss.Color("#442222"), // dark red
		Danger:         lipgloss.Color("#FF0000"), // bright red
		Success:        lipgloss.Color("#FFAA00"), // orange (caution even for success)
		Warning:        lipgloss.Color("#FF6600"), // orange-red
		TextBright:     lipgloss.Color("#FFFFFF"),
		TextNormal:     lipgloss.Color("#FFCCCC"), // light red tint
		TextDim:        lipgloss.Color("#AA6666"),
		SyntaxString:   lipgloss.Color("#FFAA88"),
		SyntaxNumber:   lipgloss.Color("#FF8888"),
		SyntaxKeyword:  lipgloss.Color("#FF6666"),
		SyntaxNull:     lipgloss.Color("#AA6666"),
		SyntaxBoolean:  lipgloss.Color("#FFCC88"),
		SyntaxDatetime: lipgloss.Color("#FFAA66"),
		SyntaxFunction: lipgloss.Color("#FFCC66"),
		SyntaxComment:  lipgloss.Color("#AA6666"),
		SyntaxOperator: lipgloss.Color("#FF8888"),
	},

	"forest": {
		Name:           "forest",
		Description:    "Calming green forest theme",
		Primary:        lipgloss.Color("#4CAF50"), // green
		Secondary:      lipgloss.Color("#2E4A32"), // dark green
		Danger:         lipgloss.Color("#FF5252"), // red
		Success:        lipgloss.Color("#81C784"), // light green
		Warning:        lipgloss.Color("#FFD54F"), // amber
		TextBright:     lipgloss.Color("#E8F5E9"), // light green white
		TextNormal:     lipgloss.Color("#A5D6A7"),
		TextDim:        lipgloss.Color("#5D7A5F"),
		SyntaxString:   lipgloss.Color("#AED581"), // light green
		SyntaxNumber:   lipgloss.Color("#CE93D8"), // purple
		SyntaxKeyword:  lipgloss.Color("#4DB6AC"), // teal
		SyntaxNull:     lipgloss.Color("#5D7A5F"),
		SyntaxBoolean:  lipgloss.Color("#80DEEA"), // cyan
		SyntaxDatetime: lipgloss.Color("#FFD54F"), // amber
		SyntaxFunction: lipgloss.Color("#80DEEA"), // cyan
		SyntaxComment:  lipgloss.Color("#5D7A5F"),
		SyntaxOperator: lipgloss.Color("#A5D6A7"),
	},

	"ocean": {
		Name:           "ocean",
		Description:    "Deep ocean blue theme",
		Primary:        lipgloss.Color("#0288D1"), // light blue
		Secondary:      lipgloss.Color("#01579B"), // dark blue
		Danger:         lipgloss.Color("#FF5252"), // red
		Success:        lipgloss.Color("#00E676"), // green
		Warning:        lipgloss.Color("#FFC107"), // amber
		TextBright:     lipgloss.Color("#E1F5FE"), // light blue white
		TextNormal:     lipgloss.Color("#81D4FA"),
		TextDim:        lipgloss.Color("#4A6572"),
		SyntaxString:   lipgloss.Color("#80CBC4"), // teal
		SyntaxNumber:   lipgloss.Color("#F48FB1"), // pink
		SyntaxKeyword:  lipgloss.Color("#82B1FF"), // blue
		SyntaxNull:     lipgloss.Color("#4A6572"),
		SyntaxBoolean:  lipgloss.Color("#84FFFF"), // cyan
		SyntaxDatetime: lipgloss.Color("#FFE082"), // amber light
		SyntaxFunction: lipgloss.Color("#84FFFF"), // cyan
		SyntaxComment:  lipgloss.Color("#4A6572"),
		SyntaxOperator: lipgloss.Color("#82B1FF"), // blue
	},
}

// DefaultTheme is the theme used when none is specified
var DefaultTheme = Themes["default"]

// GetTheme returns a theme by name, or the default if not found
func GetTheme(name string) Theme {
	if name == "" {
		return DefaultTheme
	}
	if theme, ok := Themes[name]; ok {
		return theme
	}
	return DefaultTheme
}

// ThemeNames returns a sorted list of available theme names
func ThemeNames() []string {
	// Return in a nice order (default first, then alphabetical, production last)
	return []string{
		"default",
		"catppuccin",
		"dracula",
		"forest",
		"gruvbox",
		"monokai",
		"nord",
		"ocean",
		"solarized",
		"tokyo-night",
		"production",
	}
}
