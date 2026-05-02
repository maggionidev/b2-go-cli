package main

import "github.com/charmbracelet/lipgloss"

var (
	cyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	bold    = lipgloss.NewStyle().Bold(true)
	dim     = lipgloss.NewStyle().Faint(true)
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	magenta = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Bold(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("86")).
			Padding(0, 2)

	successPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("82")).
			Padding(0, 2)

	errorPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(0, 2)
)