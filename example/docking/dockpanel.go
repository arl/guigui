// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import "github.com/guigui-gui/guigui"

// DockPanel is a unit of dockable content. Title labels the panel's tab, and
// Content is the widget shown while the panel is the active tab of its group.
type DockPanel struct {
	Title   string
	Content guigui.Widget
}
