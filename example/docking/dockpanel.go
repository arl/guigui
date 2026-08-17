// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

var dockPanelEventDragStart guigui.EventKey = guigui.GenerateEventKey()

// DockPanel is a single dockable panel: a title bar above a content widget.
// Unpinning collapses the panel to its title bar so the sibling docked widget
// can take the freed space. Dragging the title bar moves the panel (see
// DockingLayout).
type DockPanel struct {
	guigui.DefaultWidget

	title   string
	content guigui.Widget

	// Pinned reports whether the panel is expanded. An unpinned panel shows
	// only its pin button.
	Pinned bool

	titleText basicwidget.Text
	pinButton basicwidget.Button
}

// OnDragStart registers a handler invoked when the user presses the panel's
// title bar to begin dragging it. The origin is the cursor position at the
// press, in screen coordinates.
func (p *DockPanel) OnDragStart(f func(context *guigui.Context, origin image.Point)) {
	guigui.SetEventHandler(p, dockPanelEventDragStart, f)
}

func (p *DockPanel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&p.pinButton)
	if p.Pinned {
		adder.AddWidget(&p.titleText)
		if p.content != nil {
			adder.AddWidget(p.content)
		}
	}

	if p.Pinned {
		p.titleText.SetValue(p.title)
		var style basicwidget.TextStyle
		style.SetBold(true)
		p.titleText.SetBaseStyle(&style)
		p.titleText.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	}

	p.pinButton.SetText(pinLabel(p.Pinned))
	p.pinButton.OnDown(func(context *guigui.Context) {
		p.Pinned = !p.Pinned
	})
	return nil
}

func pinLabel(pinned bool) string {
	if pinned {
		return "-"
	}
	return "+"
}

func (p *DockPanel) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !p.Pinned || !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return guigui.HandleInputResult{}
	}
	u := basicwidget.UnitSize(context)
	b := widgetBounds.Bounds()
	// The title bar is the top strip; the pin button occupies its trailing end,
	// so it is excluded from the drag handle.
	dragHandle := image.Rectangle{
		Min: b.Min,
		Max: image.Pt(b.Max.X-u, b.Min.Y+u),
	}
	if cursor := image.Pt(ebiten.CursorPosition()); cursor.In(dragHandle) {
		guigui.DispatchEvent(p, dockPanelEventDragStart, cursor)
		return guigui.HandleInputByWidget(p)
	}
	return guigui.HandleInputResult{}
}

func (p *DockPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	b := widgetBounds.Bounds()

	if !p.Pinned {
		layouter.LayoutWidget(&p.pinButton, b)
		return
	}

	titleBounds := image.Rectangle{
		Min: b.Min,
		Max: image.Pt(b.Max.X, b.Min.Y+u),
	}
	pinBounds := image.Rectangle{
		Min: image.Pt(titleBounds.Max.X-u, b.Min.Y),
		Max: image.Pt(titleBounds.Max.X, titleBounds.Max.Y),
	}
	textBounds := image.Rectangle{
		Min: image.Pt(b.Min.X+u/2, b.Min.Y),
		Max: image.Pt(pinBounds.Min.X, titleBounds.Max.Y),
	}
	layouter.LayoutWidget(&p.titleText, textBounds)
	layouter.LayoutWidget(&p.pinButton, pinBounds)

	if p.content != nil {
		layouter.LayoutWidget(p.content, image.Rectangle{
			Min: image.Pt(b.Min.X, titleBounds.Max.Y),
			Max: b.Max,
		})
	}
}
