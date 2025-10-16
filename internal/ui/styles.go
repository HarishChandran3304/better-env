package ui

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
)

var (
    accentColor    = lipgloss.AdaptiveColor{Light: "#2F3EC9", Dark: "#9EA8FF"}
    textColor      = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#E5E7EB"}
    subtleColor    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#94A3B8"}
    successColor   = lipgloss.AdaptiveColor{Light: "#0F766E", Dark: "#34D399"}
    warningColor   = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
    errorColor     = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
    codeBlockColor = lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#111827"}
)

var (
    baseStyle      = lipgloss.NewStyle().Foreground(textColor)
    titleStyle     = baseStyle.Copy().Foreground(accentColor).Bold(true)
    sectionStyle   = titleStyle.Copy().MarginTop(1)
    bodyStyle      = baseStyle.Copy()
    dimStyle       = baseStyle.Copy().Foreground(subtleColor)
    successStyle   = baseStyle.Copy().Foreground(successColor).Bold(true)
    warningStyle   = baseStyle.Copy().Foreground(warningColor).Bold(true)
    errorStyle     = baseStyle.Copy().Foreground(errorColor).Bold(true)
    infoStyle      = baseStyle.Copy().Foreground(accentColor)
    bulletStyle    = baseStyle.Copy().Foreground(accentColor).Bold(true)
    keyStyle       = baseStyle.Copy().Foreground(accentColor).Bold(true)
    valueStyle     = baseStyle.Copy()
    promptStyle    = baseStyle.Copy().Foreground(accentColor).Bold(true)
    dividerStyle   = baseStyle.Copy().Foreground(subtleColor)
    codeBlockStyle = baseStyle.Copy().Background(codeBlockColor).Padding(0, 1)
)

// Title renders a prominent title for the CLI output.
func Title(text string) string {
    return titleStyle.Render(text)
}

// Section renders a section heading with breathing room.
func Section(text string) string {
    return sectionStyle.Render(text)
}

// Body renders neutral body copy.
func Body(text string) string {
    return bodyStyle.Render(text)
}

// Dim renders de-emphasised helper text.
func Dim(text string) string {
    return dimStyle.Render(text)
}

// Success renders a success message with a subtle checkmark accent.
func Success(text string) string {
    return successStyle.Render("✔ " + text)
}

// Warning renders a warning message with an accent icon.
func Warning(text string) string {
    return warningStyle.Render("⚠ " + text)
}

// Error renders an error message with an accent icon.
func Error(text string) string {
    return errorStyle.Render("✖ " + text)
}

// Info renders informational text with an accent.
func Info(text string) string {
    return infoStyle.Render("ℹ " + text)
}

// BulletList renders items as a professional bullet list.
func BulletList(items []string) string {
    if len(items) == 0 {
        return ""
    }

    bullet := bulletStyle.Render("•")
    lines := make([]string, 0, len(items))
    for _, item := range items {
        lines = append(lines, fmt.Sprintf("%s %s", bullet, bodyStyle.Render(item)))
    }
    return strings.Join(lines, "\n")
}

// NumberedList renders items as an ordered list starting at 1.
func NumberedList(items []string) string {
    if len(items) == 0 {
        return ""
    }

    lines := make([]string, 0, len(items))
    for i, item := range items {
        lines = append(lines, fmt.Sprintf("%2d. %s", i+1, bodyStyle.Render(item)))
    }
    return strings.Join(lines, "\n")
}

// KeyValue renders a key/value pair with a subtle accent.
func KeyValue(key, value string) string {
    return fmt.Sprintf("%s%s%s", keyStyle.Render(key), dimStyle.Render(" = "), valueStyle.Render(value))
}

// Prompt styles prompts shown before interactive input.
func Prompt(label string) string {
    return promptStyle.Render(label)
}

// CodeBlock renders text within a subdued code block treatment.
func CodeBlock(lines ...string) string {
    return codeBlockStyle.Render(strings.Join(lines, "\n"))
}

// Divider renders a subtle horizontal rule.
func Divider(width int) string {
    if width <= 0 {
        width = 38
    }
    return dividerStyle.Render(strings.Repeat("─", width))
}
