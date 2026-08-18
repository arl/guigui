// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
)

// Pane is a single collapsible panel in a PaneView.
type Pane struct {
	guigui.DefaultWidget

	title          string
	content        guigui.Widget
	expanded       bool
	headerVisible  bool
	headerSize     int
	minBodySize    int
	maxBodySize    int
	onExpandChange func(expanded bool)
}

// PaneView is a vertical stack of collapsible panes with headers.
// Similar to dockview's PaneView: each pane has a header (always visible)
// and a body that can be collapsed/expanded. Panes can be reordered by
// dragging their headers.
type PaneView struct {
	guigui.DefaultWidget

	panes      []*Pane
	dividers   []paneDivider
	dragIndex  int
	dragOffset int
}

type paneDivider struct {
	bounds image.Rectangle
	index  int // divider between pane index and index+1
}

const paneDividerThickness = 4

// NewPane creates a new pane for use in a PaneView.
func NewPane(title string, content guigui.Widget) *Pane {
	return &Pane{
		title:         title,
		content:       content,
		expanded:      true,
		headerVisible: true,
		headerSize:    28,
		minBodySize:   50,
		maxBodySize:   10000,
	}
}

func (p *Pane) SetExpanded(expanded bool) {
	if p.expanded == expanded {
		return
	}
	p.expanded = expanded
	if p.onExpandChange != nil {
		p.onExpandChange(expanded)
	}
}

func (p *Pane) IsExpanded() bool {
	return p.expanded
}

func (p *Pane) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(p.content)
	return nil
}

func (p *Pane) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	if p.expanded {
		layouter.LayoutWidget(p.content, widgetBounds.Bounds())
	}
}

func (p *Pane) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	if !p.expanded {
		fixedW, _ := constraints.FixedWidth()
		return image.Pt(fixedW, p.headerSize)
	}
	size := p.content.Measure(context, constraints)
	size.Y += p.headerSize
	return size
}

func (p *Pane) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	b := widgetBounds.Bounds()

	// Header background
	var headerBg color.RGBA
	if context.ColorMode() == ebiten.ColorModeLight {
		headerBg = color.RGBA{0xe0, 0xe0, 0xe0, 0xff}
	} else {
		headerBg = color.RGBA{0x30, 0x30, 0x30, 0xff}
	}
	vector.DrawFilledRect(dst, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(p.headerSize), headerBg, false)

	// Header separator
	var sepColor color.RGBA
	if context.ColorMode() == ebiten.ColorModeLight {
		sepColor = color.RGBA{0xaa, 0xaa, 0xaa, 0xff}
	} else {
		sepColor = color.RGBA{0x55, 0x55, 0x55, 0xff}
	}
	vector.StrokeLine(dst, float32(b.Min.X), float32(b.Min.Y+p.headerSize), float32(b.Max.X), float32(b.Min.Y+p.headerSize), 1, sepColor, false)

	// Expand/collapse indicator (simple rect for now)
	indicatorX := b.Min.X + 6
	indicatorY := b.Min.Y + (p.headerSize-16)/2
	var triColor color.RGBA
	if context.ColorMode() == ebiten.ColorModeLight {
		triColor = color.RGBA{0x33, 0x33, 0x33, 0xff}
	} else {
		triColor = color.RGBA{0xcc, 0xcc, 0xcc, 0xff}
	}
	if p.expanded {
		// Down triangle (simplified as rect)
		vector.DrawFilledRect(dst, float32(indicatorX+4), float32(indicatorY+4), 8, 8, triColor, false)
	} else {
		// Right triangle (simplified as rect)
		vector.DrawFilledRect(dst, float32(indicatorX+4), float32(indicatorY+4), 8, 8, triColor, false)
	}
}

func (p *Pane) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && widgetBounds.IsHitAtCursor() {
		cursor := image.Pt(ebiten.CursorPosition())
		// Click on header area toggles expansion
		if cursor.Y < widgetBounds.Bounds().Min.Y+p.headerSize {
			p.SetExpanded(!p.expanded)
			return guigui.HandleInputByWidget(p)
		}
	}
	return guigui.HandleInputResult{}
}

func (p *Pane) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	cursor := image.Pt(ebiten.CursorPosition())
	if cursor.Y < widgetBounds.Bounds().Min.Y+p.headerSize {
		return ebiten.CursorShapePointer, true
	}
	return ebiten.CursorShapeDefault, false
}

// PaneView implementation

func (pv *PaneView) AddPane(pane *Pane) {
	pv.panes = append(pv.panes, pane)
}

func (pv *PaneView) RemovePane(index int) *Pane {
	if index < 0 || index >= len(pv.panes) {
		return nil
	}
	pane := pv.panes[index]
	pv.panes = append(pv.panes[:index], pv.panes[index+1:]...)
	return pane
}

func (pv *PaneView) MovePane(from, to int) {
	if from < 0 || from >= len(pv.panes) || to < 0 || to >= len(pv.panes) || from == to {
		return
	}
	pane := pv.panes[from]
	pv.panes = append(pv.panes[:from], pv.panes[from+1:]...)
	pv.panes = append(pv.panes[:to], append([]*Pane{pane}, pv.panes[to:]...)...)
}

func (pv *PaneView) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	for _, pane := range pv.panes {
		adder.AddWidget(pane)
	}
	return nil
}

func (pv *PaneView) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	pv.dividers = slices.Delete(pv.dividers, 0, len(pv.dividers))
	b := widgetBounds.Bounds()
	y := b.Min.Y

	for i, pane := range pv.panes {
		paneHeight := pane.headerSize
		if pane.expanded {
			// For expanded panes, give them their preferred size within remaining space
			// For simplicity, distribute remaining space proportionally
			remainingPanes := len(pv.panes) - i
			remainingHeight := b.Max.Y - y
			// Each expanded pane gets at least minBodySize
			minNeeded := 0
			for j := i; j < len(pv.panes); j++ {
				if pv.panes[j].expanded {
					minNeeded += pv.panes[j].minBodySize + pv.panes[j].headerSize
				} else {
					minNeeded += pv.panes[j].headerSize
				}
			}
			if remainingHeight >= minNeeded && remainingPanes > 0 {
				// Distribute extra space
				extra := remainingHeight - minNeeded
				bodySize := pane.minBodySize + extra/remainingPanes
				if bodySize > pane.maxBodySize {
					bodySize = pane.maxBodySize
				}
				paneHeight += bodySize
			} else {
				paneHeight += pane.minBodySize
			}
		}
		// Clamp to available space
		if y+paneHeight > b.Max.Y {
			paneHeight = b.Max.Y - y
		}

		paneBounds := image.Rect(b.Min.X, y, b.Max.X, y+paneHeight)
		layouter.LayoutWidget(pane, paneBounds)

		// Add divider after this pane (except last)
		if i < len(pv.panes)-1 {
			dividerY := y + paneHeight
			pv.dividers = append(pv.dividers, paneDivider{
				bounds: image.Rect(b.Min.X, dividerY, b.Max.X, dividerY+paneDividerThickness),
				index:  i,
			})
			y = dividerY + paneDividerThickness
		} else {
			y += paneHeight
		}
	}
}

func (pv *PaneView) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	cursor := image.Pt(ebiten.CursorPosition())

	// Check divider drag
	if pv.dragIndex >= 0 {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			// Update divider position
			if pv.dragIndex < len(pv.dividers) {
				newY := cursor.Y - pv.dragOffset
				// Clamp between adjacent panes' min sizes
				// For now just allow free movement
				_ = newY // In a full implementation, this would resize panes
			}
			return guigui.HandleInputByWidget(pv)
		}
		pv.dragIndex = -1
		return guigui.HandleInputResult{}
	}

	// Check for divider press
	for i, div := range pv.dividers {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && cursor.In(div.bounds) {
			pv.dragIndex = i
			pv.dragOffset = cursor.Y - div.bounds.Min.Y
			return guigui.HandleInputByWidget(pv)
		}
	}

	// Check for pane header drag (reorder)
	for i, pane := range pv.panes {
		b := widgetBounds.Bounds()
		paneTop := b.Min.Y
		for j := 0; j < i; j++ {
			paneTop += pv.panes[j].headerSize
			if pv.panes[j].expanded {
				paneTop += pv.panes[j].minBodySize
			}
			if j < len(pv.dividers) {
				paneTop += paneDividerThickness
			}
		}
		headerRect := image.Rect(b.Min.X, paneTop, b.Max.X, paneTop+pane.headerSize)
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && cursor.In(headerRect) {
			// Start drag for reorder
			// For now just toggle expansion on click; drag-reorder would need more state
			pane.SetExpanded(!pane.expanded)
			return guigui.HandleInputByWidget(pv)
		}
	}

	return guigui.HandleInputResult{}
}

func (pv *PaneView) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	cursor := image.Pt(ebiten.CursorPosition())

	for _, div := range pv.dividers {
		if cursor.In(div.bounds) {
			return ebiten.CursorShapeNSResize, true
		}
	}

	return ebiten.CursorShapeDefault, false
}
