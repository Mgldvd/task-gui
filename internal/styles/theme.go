package styles

import "github.com/charmbracelet/lipgloss"

// Theme centralizes all lipgloss styles used by the TUI.
// Only modify styles here to keep presentation concerns isolated.
type Theme struct {
	Title         lipgloss.Style
	TaskName      lipgloss.Style
	Command       lipgloss.Style
	Description   lipgloss.Style
	Selected      lipgloss.Style
	SelectedWire  lipgloss.Style
	Border        lipgloss.Style
	Help          lipgloss.Style
	Status        lipgloss.Style
	Output        lipgloss.Style
	Error         lipgloss.Style
	HeaderBox     lipgloss.Style
	CommandBox    lipgloss.Style
	ContentBox    lipgloss.Style
	SearchBox     lipgloss.Style
	FooterBox     lipgloss.Style
	TabActive     lipgloss.Style
	TabInactive   lipgloss.Style
	TabsContainer lipgloss.Style
	TabArrow      lipgloss.Style
	AppTitle      lipgloss.Style
	AppContainer  lipgloss.Style
	Gradient      lipgloss.Style
	Highlight     lipgloss.Style
	Accent        lipgloss.Style
	Logo          lipgloss.Style
}

// NewDarkTheme returns the dark color scheme.
func NewDarkTheme() Theme {
	return Theme{
		AppTitle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A855F7")).Padding(0, 4),
		AppContainer: lipgloss.NewStyle().Padding(1, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#8B5CF6")),

		HeaderBox: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E2E8F0")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#A855F7")).Padding(1, 2).Margin(0, 0, 1, 0),

		TabActive:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EC4899")).Padding(0, 3).Margin(0, 1),
		TabInactive:   lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Padding(0, 3).Margin(0, 1),
		TabsContainer: lipgloss.NewStyle().Padding(0, 1).Margin(0, 0, 1, 0).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(lipgloss.Color("#7C3AED")),
		TabArrow:      lipgloss.NewStyle().Foreground(lipgloss.Color("#EC4899")).Bold(true),

		CommandBox:   lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B21A8")).Padding(0, 1),
		Selected:     lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#EC4899")).Padding(0, 1),
		// SelectedWire now only changes the border color (not the text) so inner highlight can target just the task name.
		SelectedWire: lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#EC4899")).Padding(0, 1),

		ContentBox: lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B21A8")).Padding(1, 2).Margin(0, 0, 1, 0),
		SearchBox:  lipgloss.NewStyle().Foreground(lipgloss.Color("#C084FC")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#A855F7")).Padding(0, 2).Margin(0, 0, 1, 0),
		FooterBox:  lipgloss.NewStyle().Foreground(lipgloss.Color("#A0AEC0")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B21A8")).Padding(0, 2, 0, 2).Margin(1, 0, 0, 0),

		Title:       lipgloss.NewStyle().Foreground(lipgloss.Color("#F7FAFC")).Bold(true),
		TaskName:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true),
		Command:     lipgloss.NewStyle().Foreground(lipgloss.Color("#68D391")).Italic(true),
		Description: lipgloss.NewStyle().Foreground(lipgloss.Color("#A0AEC0")),
		Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")),
		Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("#68D391")).Bold(true),
		Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("#FC8181")).Bold(true),
		Output:      lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")),
		Border:      lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5568")),

		Gradient:  lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")),
		Highlight: lipgloss.NewStyle().Foreground(lipgloss.Color("#EC4899")),
		Accent:    lipgloss.NewStyle().Foreground(lipgloss.Color("#DDD6FE")),
		Logo:      lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7")).Bold(true),
	}
}

// NewLightTheme returns the light color scheme.
func NewLightTheme() Theme {
	return Theme{
		AppTitle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")).Padding(0, 4),
		AppContainer: lipgloss.NewStyle().Padding(1, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#A855F7")),

		HeaderBox: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2D3748")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7C3AED")).Padding(1, 2).Margin(0, 0, 1, 0),

		TabActive:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EC4899")).Padding(0, 3).Margin(0, 1),
		TabInactive:   lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5568")).Padding(0, 3).Margin(0, 1),
		TabsContainer: lipgloss.NewStyle().Padding(0, 1).Margin(0, 0, 1, 0).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(lipgloss.Color("#C084FC")),
		TabArrow:      lipgloss.NewStyle().Foreground(lipgloss.Color("#EC4899")).Bold(true),

		CommandBox:   lipgloss.NewStyle().Foreground(lipgloss.Color("#2D3748")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#A855F7")).Padding(0, 1),
		Selected:     lipgloss.NewStyle().Foreground(lipgloss.Color("#2D3748")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#EC4899")).Padding(0, 1),
		SelectedWire: lipgloss.NewStyle().Foreground(lipgloss.Color("#2D3748")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#EC4899")).Padding(0, 1),

		ContentBox: lipgloss.NewStyle().Foreground(lipgloss.Color("#2D3748")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#A855F7")).Padding(1, 2).Margin(0, 0, 1, 0),
		SearchBox:  lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#8B5CF6")).Padding(0, 2).Margin(0, 0, 1, 0),
		FooterBox:  lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5568")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#A855F7")).Padding(0, 2, 0, 2).Margin(1, 0, 0, 0),

		Title:       lipgloss.NewStyle().Foreground(lipgloss.Color("#1A202C")).Bold(true),
		TaskName:    lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Bold(true),
		Command:     lipgloss.NewStyle().Foreground(lipgloss.Color("#047857")).Italic(true),
		Description: lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5568")),
		Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("#718096")),
		Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("#059669")).Bold(true),
		Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true),
		Output:      lipgloss.NewStyle().Foreground(lipgloss.Color("#1A202C")),
		Border:      lipgloss.NewStyle().Foreground(lipgloss.Color("#A0AEC0")),

		Gradient:  lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")),
		Highlight: lipgloss.NewStyle().Foreground(lipgloss.Color("#EC4899")),
		Accent:    lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7")),
		Logo:      lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true),
	}
}
