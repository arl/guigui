// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// edgeSide identifies a vertical sticky edge on the root.
type edgeSide int

const (
	edgeSideNone edgeSide = iota
	edgeSideLeft
	edgeSideRight
)

// edgeOpenWidth returns the total bar width when expanded or pinned.
func edgeOpenWidth(u int) int { return u * 10 }

// edgeBar is a vertical sticky tab group docked on the root's left or right
// edge. Its group renders a vertical tab strip plus the selected content.
type edgeBar struct {
	guigui.DefaultWidget

	group    *DockGroup
	side     edgeSide
	expanded bool
	pinned   bool

	pinRect image.Rectangle

	// Pending tab press, used to distinguish a click from a drag.
	pressPanel *DockPanel
	pressPos   image.Point

	onDragStart func(panel *DockPanel, cursor image.Point)
}

func newEdgeBar(side edgeSide) *edgeBar {
	return &edgeBar{
		side: side,
		group: &DockGroup{
			vertical:     true,
			stripOnRight: side == edgeSideRight,
		},
	}
}

func (b *edgeBar) isOpen() bool { return b.expanded || b.pinned }

func (b *edgeBar) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	b.group.collapsed = !b.isOpen()
	b.group.onTabClick = func(panel *DockPanel, cursor image.Point) {
		b.pressPanel = panel
		b.pressPos = cursor
	}
	adder.AddWidget(b.group)
	return nil
}

func (b *edgeBar) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	bnds := widgetBounds.Bounds()

	// Reserve the top stripWidth of the strip for the pin toggle.
	stripTop := bnds.Min.Y + u
	sw := stripWidth(u)
	if b.side == edgeSideLeft {
		b.pinRect = image.Rectangle{Min: bnds.Min, Max: image.Pt(bnds.Min.X+sw, stripTop)}
	} else {
		b.pinRect = image.Rectangle{Min: image.Pt(bnds.Max.X-sw, bnds.Min.Y), Max: image.Pt(bnds.Max.X, stripTop)}
	}

	groupBounds := image.Rectangle{Min: image.Pt(bnds.Min.X, stripTop), Max: bnds.Max}
	layouter.LayoutWidget(b.group, groupBounds)
}

func (b *edgeBar) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	cursor := image.Pt(ebiten.CursorPosition())
	u := basicwidget.UnitSize(context)

	// Resolve a pending tab press: click (expand/collapse/select) vs drag (move out).
	if b.pressPanel != nil {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			if absInt(cursor.X-b.pressPos.X)+absInt(cursor.Y-b.pressPos.Y) > u/2 {
				panel := b.pressPanel
				b.pressPanel = nil
				if b.onDragStart != nil {
					b.onDragStart(panel, cursor)
				}
				return guigui.HandleInputByWidget(b)
			}
			return guigui.HandleInputResult{}
		}
		b.clickTab(b.pressPanel)
		b.pressPanel = nil
		return guigui.HandleInputByWidget(b)
	}

	// Pin toggle.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && cursor.In(b.pinRect) {
		b.pinned = !b.pinned
		guigui.RequestRebuild()
		return guigui.HandleInputByWidget(b)
	}

	return guigui.HandleInputResult{}
}

// clickTab selects the tab and expands/collapses the bar as appropriate.
func (b *edgeBar) clickTab(panel *DockPanel) {
	g := b.group
	idx := -1
	for i, p := range g.panels {
		if p == panel {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	switch {
	case !b.isOpen():
		g.selected = idx
		b.expanded = true
	case b.pinned:
		g.selected = idx
	case g.selected == idx:
		b.expanded = false
	default:
		g.selected = idx
	}
	guigui.RequestRebuild()
}

// Tick collapses an unpinned, expanded bar when the user presses outside it.
func (b *edgeBar) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if b.pinned || !b.expanded {
		return nil
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if !image.Pt(ebiten.CursorPosition()).In(widgetBounds.Bounds()) {
			b.expanded = false
			guigui.RequestRebuild()
		}
	}
	return nil
}

func (b *edgeBar) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if image.Pt(ebiten.CursorPosition()).In(b.pinRect) {
		return ebiten.CursorShapePointer, true
	}
	return ebiten.CursorShapeDefault, false
}

func (b *edgeBar) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	u := basicwidget.UnitSize(context)
	bnds := widgetBounds.Bounds()
	sw := stripWidth(u)

	var bg color.RGBA
	if context.ColorMode() == ebiten.ColorModeLight {
		bg = color.RGBA{0xd8, 0xd8, 0xd8, 0xff}
	} else {
		bg = color.RGBA{0x28, 0x28, 0x28, 0xff}
	}
	if b.side == edgeSideLeft {
		vector.DrawFilledRect(dst, float32(bnds.Min.X), float32(bnds.Min.Y), float32(sw), float32(bnds.Dy()), bg, false)
	} else {
		vector.DrawFilledRect(dst, float32(bnds.Max.X-sw), float32(bnds.Min.Y), float32(sw), float32(bnds.Dy()), bg, false)
	}

	// Pin toggle: filled when pinned, hollow otherwise.
	var pin color.RGBA
	if b.pinned {
		if context.ColorMode() == ebiten.ColorModeLight {
			pin = color.RGBA{0x1a, 0x6e, 0xf5, 0xff}
		} else {
			pin = color.RGBA{0x5a, 0x9e, 0xf5, 0xff}
		}
	} else {
		if context.ColorMode() == ebiten.ColorModeLight {
			pin = color.RGBA{0x80, 0x80, 0x80, 0xff}
		} else {
			pin = color.RGBA{0x80, 0x80, 0x80, 0xff}
		}
	}
	pr := b.pinRect
	cx := float32(pr.Min.X + pr.Dx()/2)
	cy := float32(pr.Min.Y + pr.Dy()/2)
	r := float32(u / 4)
	vector.DrawFilledCircle(dst, cx, cy, r, pin, true)
	vector.StrokeCircle(dst, cx, cy, r, 1, color.RGBA{0, 0, 0, 0x66}, false)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
