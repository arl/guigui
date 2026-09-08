// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package datepicker

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
)

// Approximations of basicwidget's internal theme colors, which have no public
// equivalent: a flat accent fill, white text on top of it, and the secondary
// control color as a stand-in for a hovered item's background.

func accentColor(ebiten.ColorMode) color.Color {
	return basicwidget.AccentTintColor()
}

func textOnAccentColor(ebiten.ColorMode) color.Color {
	return color.White
}

func itemHoveredBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return basicwidgetdraw.ControlSecondaryColor(colorMode, true)
}
