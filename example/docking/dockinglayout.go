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
	"github.com/guigui-gui/guigui/basicwidget"
)

// DockNode is a node in the docking tree. Exactly one of panel or split is
// non-nil.
type DockNode struct {
	panel *DockPanel
	split *DockSplit
}

// DockSplit divides its space between two child nodes, separated by a
// draggable divider.
type DockSplit struct {
	direction guigui.LayoutDirection
	// ratio is the fraction of the available extent given to first, in
	// [0, 1].
	ratio  float64
	first  *DockNode
	second *DockNode
}

type dockDivider struct {
	split  *DockSplit
	bounds image.Rectangle
	// available is the total extent along the split axis, minus the divider
	// thickness, at the time of the last layout.
	available int
}

// DockingLayout arranges dockable panels in a nestable split tree whose
// dividers can be dragged to resize the panels.
type DockingLayout struct {
	guigui.DefaultWidget

	root *DockNode

	dividers []dockDivider

	dragging       *DockSplit
	dragOrigin     image.Point
	dragStartRatio float64
	dragAvailable  int
}

// SetRoot sets the root of the docking tree.
func (d *DockingLayout) SetRoot(root *DockNode) {
	d.root = root
}

func (d *DockingLayout) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if d.root != nil {
		d.addNode(adder, d.root)
	}
	return nil
}

func (d *DockingLayout) addNode(adder *guigui.ChildAdder, node *DockNode) {
	if node.panel != nil {
		adder.AddWidget(node.panel)
		return
	}
	if node.split != nil {
		d.addNode(adder, node.split.first)
		d.addNode(adder, node.split.second)
	}
}

func (d *DockingLayout) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	d.dividers = slices.Delete(d.dividers, 0, len(d.dividers))
	if d.root != nil {
		d.layoutNode(context, d.root, widgetBounds.Bounds(), layouter)
	}
}

func (d *DockingLayout) dividerThickness(context *guigui.Context) int {
	return max(6, basicwidget.UnitSize(context)/4)
}

// collapsedExtent reports the fixed extent a collapsed panel takes along the
// split axis, and whether node is a collapsed panel.
func collapsedExtent(context *guigui.Context, node *DockNode) (int, bool) {
	if node.panel != nil && !node.panel.Pinned {
		return basicwidget.UnitSize(context), true
	}
	return 0, false
}

// splitExtents computes the extents of the two children along the split axis,
// honoring collapsed panels before falling back to the ratio.
func (d *DockingLayout) splitExtents(context *guigui.Context, split *DockSplit, available int) (int, int) {
	if ce, ok := collapsedExtent(context, split.first); ok {
		return ce, available - ce
	}
	if ce, ok := collapsedExtent(context, split.second); ok {
		return available - ce, ce
	}
	minExtent := basicwidget.UnitSize(context)
	first := int(float64(available) * split.ratio)
	second := available - first
	if first < minExtent {
		first, second = minExtent, available-minExtent
	}
	if second < minExtent {
		first, second = available-minExtent, minExtent
	}
	return first, second
}

func (d *DockingLayout) layoutNode(context *guigui.Context, node *DockNode, bounds image.Rectangle, layouter *guigui.ChildLayouter) {
	if node.panel != nil {
		layouter.LayoutWidget(node.panel, bounds)
		return
	}
	split := node.split
	if split == nil {
		return
	}

	t := d.dividerThickness(context)
	var first, divider, second image.Rectangle
	var available int
	switch split.direction {
	case guigui.LayoutDirectionHorizontal:
		available = bounds.Dx() - t
		f, _ := d.splitExtents(context, split, available)
		first = image.Rectangle{
			Min: bounds.Min,
			Max: image.Pt(bounds.Min.X+f, bounds.Max.Y),
		}
		divider = image.Rectangle{
			Min: image.Pt(bounds.Min.X+f, bounds.Min.Y),
			Max: image.Pt(bounds.Min.X+f+t, bounds.Max.Y),
		}
		second = image.Rectangle{
			Min: image.Pt(bounds.Min.X+f+t, bounds.Min.Y),
			Max: bounds.Max,
		}
	case guigui.LayoutDirectionVertical:
		available = bounds.Dy() - t
		f, _ := d.splitExtents(context, split, available)
		first = image.Rectangle{
			Min: bounds.Min,
			Max: image.Pt(bounds.Max.X, bounds.Min.Y+f),
		}
		divider = image.Rectangle{
			Min: image.Pt(bounds.Min.X, bounds.Min.Y+f),
			Max: image.Pt(bounds.Max.X, bounds.Min.Y+f+t),
		}
		second = image.Rectangle{
			Min: image.Pt(bounds.Min.X, bounds.Min.Y+f+t),
			Max: bounds.Max,
		}
	}

	d.dividers = append(d.dividers, dockDivider{
		split:     split,
		bounds:    divider,
		available: available,
	})
	d.layoutNode(context, split.first, first, layouter)
	d.layoutNode(context, split.second, second, layouter)
}

func (d *DockingLayout) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	cursor := image.Pt(ebiten.CursorPosition())

	if d.dragging != nil {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			d.updateRatio(cursor)
			return guigui.HandleInputByWidget(d)
		}
		d.dragging = nil
		return guigui.HandleInputResult{}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for i := range d.dividers {
			if cursor.In(d.dividers[i].bounds) {
				d.dragging = d.dividers[i].split
				d.dragOrigin = cursor
				d.dragStartRatio = d.dividers[i].split.ratio
				d.dragAvailable = d.dividers[i].available
				return guigui.HandleInputByWidget(d)
			}
		}
	}
	return guigui.HandleInputResult{}
}

func (d *DockingLayout) updateRatio(cursor image.Point) {
	if d.dragging == nil || d.dragAvailable <= 0 {
		return
	}
	var delta int
	switch d.dragging.direction {
	case guigui.LayoutDirectionHorizontal:
		delta = cursor.X - d.dragOrigin.X
	case guigui.LayoutDirectionVertical:
		delta = cursor.Y - d.dragOrigin.Y
	}
	ratio := d.dragStartRatio + float64(delta)/float64(d.dragAvailable)
	d.dragging.ratio = min(max(ratio, 0.1), 0.9)
}

func (d *DockingLayout) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if d.dragging != nil {
		return resizeCursorFor(d.dragging.direction), true
	}
	cursor := image.Pt(ebiten.CursorPosition())
	for i := range d.dividers {
		if cursor.In(d.dividers[i].bounds) {
			return resizeCursorFor(d.dividers[i].split.direction), true
		}
	}
	return 0, false
}

func resizeCursorFor(direction guigui.LayoutDirection) ebiten.CursorShapeType {
	if direction == guigui.LayoutDirectionHorizontal {
		return ebiten.CursorShapeEWResize
	}
	return ebiten.CursorShapeNSResize
}

func (d *DockingLayout) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	var clr color.RGBA
	if context.ColorMode() == ebiten.ColorModeLight {
		clr = color.RGBA{0x80, 0x80, 0x80, 0xff}
	} else {
		clr = color.RGBA{0x40, 0x40, 0x40, 0xff}
	}
	for _, div := range d.dividers {
		b := div.bounds
		vector.DrawFilledRect(dst, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), clr, false)
	}
}
