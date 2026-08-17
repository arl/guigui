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

// DropEdge identifies a drop zone within a target panel.
type DropEdge int

const (
	dropEdgeNone DropEdge = iota
	dropEdgeLeft
	dropEdgeRight
	dropEdgeTop
	dropEdgeBottom
	dropEdgeCenter
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

type dockPanelBounds struct {
	node   *DockNode
	bounds image.Rectangle
}

// dockOverlay draws the drop-preview on top of the panels during a drag.
// It is passthrough so it does not interfere with hit testing.
type dockOverlay struct {
	guigui.DefaultWidget

	dock *DockingLayout
}

func (o *dockOverlay) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	r := o.dock.dropRect
	if r.Empty() {
		return
	}
	// A fixed, semi-transparent accent; not yet theme-integrated.
	highlight := color.RGBA{0x3b, 0x82, 0xf6, 0x66}
	vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), highlight, false)
}

// DockingLayout arranges dockable panels in a nestable split tree whose
// dividers can be dragged to resize the panels and whose title bars can be
// dragged to re-dock the panels.
type DockingLayout struct {
	guigui.DefaultWidget

	root    *DockNode
	overlay dockOverlay

	dividers    []dockDivider
	panelBounds []dockPanelBounds

	// Divider-resize drag state.
	dragging       *DockSplit
	dragOrigin     image.Point
	dragStartRatio float64
	dragAvailable  int

	// Panel-move drag state.
	dragNode *DockNode
	dropNode *DockNode
	dropEdge DropEdge
	dropRect image.Rectangle
}

// SetRoot sets the root of the docking tree.
func (d *DockingLayout) SetRoot(root *DockNode) {
	d.root = root
}

func (d *DockingLayout) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if d.root != nil {
		d.addNode(adder, d.root)
	}
	d.overlay.dock = d
	adder.AddWidget(&d.overlay)
	context.SetPassthrough(&d.overlay, true)
	return nil
}

func (d *DockingLayout) addNode(adder *guigui.ChildAdder, node *DockNode) {
	if node.panel != nil {
		adder.AddWidget(node.panel)
		node.panel.OnDragStart(func(context *guigui.Context, origin image.Point) {
			d.beginDrag(node, origin)
		})
		return
	}
	if node.split != nil {
		d.addNode(adder, node.split.first)
		d.addNode(adder, node.split.second)
	}
}

func (d *DockingLayout) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	d.dividers = slices.Delete(d.dividers, 0, len(d.dividers))
	d.panelBounds = slices.Delete(d.panelBounds, 0, len(d.panelBounds))
	if d.root != nil {
		d.layoutNode(context, d.root, widgetBounds.Bounds(), layouter)
	}
	layouter.LayoutWidget(&d.overlay, widgetBounds.Bounds())
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
		d.panelBounds = append(d.panelBounds, dockPanelBounds{node: node, bounds: bounds})
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

	// A panel drag is in flight.
	if d.dragNode != nil {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			d.updateDropTarget(cursor)
			guigui.RequestRedraw(d)
			return guigui.HandleInputByWidget(d)
		}
		d.finishDrag()
		return guigui.HandleInputByWidget(d)
	}

	// A divider resize is in flight.
	if d.dragging != nil {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			d.updateRatio(cursor)
			return guigui.HandleInputByWidget(d)
		}
		d.dragging = nil
		return guigui.HandleInputResult{}
	}

	// Start a divider resize on press.
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

// beginDrag starts moving node, called from a panel's title-bar press.
func (d *DockingLayout) beginDrag(node *DockNode, origin image.Point) {
	d.dragNode = node
	d.dropNode = nil
	d.dropEdge = dropEdgeNone
	d.dropRect = image.Rectangle{}
}

// updateDropTarget recomputes the drop zone under the cursor.
func (d *DockingLayout) updateDropTarget(cursor image.Point) {
	d.dropNode = nil
	d.dropEdge = dropEdgeNone
	d.dropRect = image.Rectangle{}
	for i := range d.panelBounds {
		pb := &d.panelBounds[i]
		if pb.node == d.dragNode || !cursor.In(pb.bounds) {
			continue
		}
		edge := dropEdgeAt(cursor, pb.bounds)
		// Center drops (tabs) are not implemented yet.
		if edge == dropEdgeCenter || edge == dropEdgeNone {
			continue
		}
		d.dropNode = pb.node
		d.dropEdge = edge
		d.dropRect = dropRectFor(pb.bounds, edge)
		return
	}
}

// finishDrag applies the pending drop, if any, and clears the drag state.
func (d *DockingLayout) finishDrag() {
	if d.dragNode != nil && d.dropNode != nil && d.dropEdge != dropEdgeNone && d.dropNode != d.dragNode {
		d.movePanel(d.dragNode, d.dropNode, d.dropEdge)
	}
	d.dragNode = nil
	d.dropNode = nil
	d.dropEdge = dropEdgeNone
	d.dropRect = image.Rectangle{}
	guigui.RequestRedraw(d)
}

// movePanel detaches dragged from its current position and docks it next to
// target on the given edge.
func (d *DockingLayout) movePanel(dragged, target *DockNode, edge DropEdge) {
	d.root = detachLeaf(d.root, dragged)
	d.root = attachLeaf(d.root, target, dragged, edge)
}

// detachLeaf removes leaf from the tree, collapsing its parent split so the
// sibling takes over. Returns the new root.
func detachLeaf(root, leaf *DockNode) *DockNode {
	if root == leaf {
		return nil
	}
	return removeLeaf(root, leaf)
}

// removeLeaf returns node's subtree with leaf removed, collapsing the split
// that directly contained it. The sibling keeps its own node identity (the
// pointer survives rather than being copied into the split slot), which
// matters when that sibling is also the target the panel is re-docked onto.
func removeLeaf(node, leaf *DockNode) *DockNode {
	if node == leaf {
		return nil
	}
	if node.split == nil {
		return node
	}
	if node.split.first == leaf {
		return node.split.second
	}
	if node.split.second == leaf {
		return node.split.first
	}
	node.split.first = removeLeaf(node.split.first, leaf)
	node.split.second = removeLeaf(node.split.second, leaf)
	return node
}

// attachLeaf replaces target with a new split holding target and dragged,
// ordered by edge. Returns the new root.
func attachLeaf(root, target, dragged *DockNode, edge DropEdge) *DockNode {
	direction, draggedFirst := splitForEdge(edge)
	newSplit := &DockNode{
		split: &DockSplit{
			direction: direction,
			ratio:     0.5,
		},
	}
	if draggedFirst {
		newSplit.split.first = dragged
		newSplit.split.second = target
	} else {
		newSplit.split.first = target
		newSplit.split.second = dragged
	}
	if root == target {
		return newSplit
	}
	replaceLeaf(root, target, newSplit)
	return root
}

// splitForEdge returns the split direction for an edge drop and whether the
// dragged panel should be the split's first child.
func splitForEdge(edge DropEdge) (guigui.LayoutDirection, bool) {
	switch edge {
	case dropEdgeLeft:
		return guigui.LayoutDirectionHorizontal, true
	case dropEdgeRight:
		return guigui.LayoutDirectionHorizontal, false
	case dropEdgeTop:
		return guigui.LayoutDirectionVertical, true
	case dropEdgeBottom:
		return guigui.LayoutDirectionVertical, false
	}
	return guigui.LayoutDirectionHorizontal, false
}

func replaceLeaf(node, target, replacement *DockNode) {
	if node.split == nil {
		return
	}
	if node.split.first == target {
		node.split.first = replacement
		return
	}
	if node.split.second == target {
		node.split.second = replacement
		return
	}
	replaceLeaf(node.split.first, target, replacement)
	replaceLeaf(node.split.second, target, replacement)
}

// dropEdgeAt returns the drop zone the cursor falls in within bounds.
func dropEdgeAt(cursor image.Point, b image.Rectangle) DropEdge {
	edgeX := b.Dx() / 3
	edgeY := b.Dy() / 3
	switch {
	case cursor.X < b.Min.X+edgeX:
		return dropEdgeLeft
	case cursor.X >= b.Max.X-edgeX:
		return dropEdgeRight
	case cursor.Y < b.Min.Y+edgeY:
		return dropEdgeTop
	case cursor.Y >= b.Max.Y-edgeY:
		return dropEdgeBottom
	default:
		return dropEdgeCenter
	}
}

func dropRectFor(b image.Rectangle, edge DropEdge) image.Rectangle {
	edgeX := b.Dx() / 3
	edgeY := b.Dy() / 3
	switch edge {
	case dropEdgeLeft:
		return image.Rectangle{Min: b.Min, Max: image.Pt(b.Min.X+edgeX, b.Max.Y)}
	case dropEdgeRight:
		return image.Rectangle{Min: image.Pt(b.Max.X-edgeX, b.Min.Y), Max: b.Max}
	case dropEdgeTop:
		return image.Rectangle{Min: b.Min, Max: image.Pt(b.Max.X, b.Min.Y+edgeY)}
	case dropEdgeBottom:
		return image.Rectangle{Min: image.Pt(b.Min.X, b.Max.Y-edgeY), Max: b.Max}
	}
	return image.Rectangle{}
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
	if d.dragNode != nil {
		return 0, false
	}
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
