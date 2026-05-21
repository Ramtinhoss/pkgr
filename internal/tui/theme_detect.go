package tui

import "github.com/muesli/termenv"

// IsDarkBackground returns true if the terminal background is dark, false if
// light, or true as a safe default if detection is unavailable.
func IsDarkBackground() bool {
	o := termenv.NewOutput(nil)
	return o.HasDarkBackground()
}

// ResolveTheme maps a config setting ("auto"/"dark"/"light") to a Theme.
// "auto" queries the terminal; unknown values fall back to dark.
func ResolveTheme(setting string) Theme {
	switch setting {
	case "dark":
		return DefaultTheme(true)
	case "light":
		return DefaultTheme(false)
	default: // "auto" or anything unrecognised
		return DefaultTheme(IsDarkBackground())
	}
}
