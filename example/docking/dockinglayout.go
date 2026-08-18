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

// DropEdge identifies a drop zone within a target group.
type DropEdge int

const (
	dropEdgeNone DropEdge = iota
	dropEdgeLeft
	dropEdgeRight
	dropEdgeTop
	dropEdgeBottom
	dropEdgeCenter
)

// DockNode is a node in the docking tree. Exactly one of group or split is
// non-nil.
type DockNode struct {
	group *DockGroup
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

type dockGroupBounds struct {
	node   *DockNode
	bounds image.Rectangle
}

// dockOverlay draws the drop-preview on top of the groups during a drag.
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

// DockingLayout arranges dockable panels in a nestable tree of groups (tabs)
// and splits. Dividers resize; dragging a tab re-docks it, either as another
// tab (center) or as a new group split onto an edge.
type DockingLayout struct {
	guigui.DefaultWidget

	root    *DockNode
	overlay dockOverlay

	dividers    []dockDivider
	groupBounds []dockGroupBounds

	// Divider-resize drag state.
	dragging       *DockSplit
	dragOrigin     image.Point
	dragStartRatio float64
	dragAvailable  int

	// Panel-move drag state.
	dragPanel  *DockPanel
	dragGroup  *DockGroup
	sourceNode *DockNode
	dropNode   *DockNode
	dropEdge   DropEdge
	dropRect   image.Rectangle
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
	if node.group != nil {
		adder.AddWidget(node.group)
		group := node.group
		group.onDragStart = func(panel *DockPanel, cursor image.Point) {
			d.beginDrag(panel, group, cursor)
		}
		return
	}
	if node.split != nil {
		d.addNode(adder, node.split.first)
		d.addNode(adder, node.split.second)
	}
}

func (d *DockingLayout) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	d.dividers = slices.Delete(d.dividers, 0, len(d.dividers))
	d.groupBounds = slices.Delete(d.groupBounds, 0, len(d.groupBounds))
	if d.root != nil {
		d.layoutNode(context, d.root, widgetBounds.Bounds(), layouter)
	}
	layouter.LayoutWidget(&d.overlay, widgetBounds.Bounds())
}

func (d *DockingLayout) dividerThickness(context *guigui.Context) int {
	return max(6, basicwidget.UnitSize(context)/4)
}

// splitExtents computes the extents of the two children along the split axis.
func (d *DockingLayout) splitExtents(context *guigui.Context, split *DockSplit, available int) (int, int) {
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
	if node.group != nil {
		layouter.LayoutWidget(node.group, bounds)
		d.groupBounds = append(d.groupBounds, dockGroupBounds{node: node, bounds: bounds})
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
	if d.dragPanel != nil {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			d.updateDropTarget(context, cursor)
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

// beginDrag starts moving panel out of group, called from a tab press.
func (d *DockingLayout) beginDrag(panel *DockPanel, group *DockGroup, cursor image.Point) {
	d.dragPanel = panel
	d.dragGroup = group
	d.sourceNode = findGroupNode(d.root, group)
	d.dropNode = nil
	d.dropEdge = dropEdgeNone
	d.dropRect = image.Rectangle{}
}

// updateDropTarget recomputes the drop zone under the cursor.
func (d *DockingLayout) updateDropTarget(context *guigui.Context, cursor image.Point) {
	d.dropNode = nil
	d.dropEdge = dropEdgeNone
	d.dropRect = image.Rectangle{}
	for i := range d.groupBounds {
		gb := &d.groupBounds[i]
		if gb.node == d.sourceNode || !cursor.In(gb.bounds) {
			continue
		}
		edge := d.dropEdgeAt(context, cursor, gb.bounds)
		if edge == dropEdgeNone {
			continue
		}
		d.dropNode = gb.node
		d.dropEdge = edge
		d.dropRect = d.dropRectFor(context, cursor, gb.bounds, edge, gb.node.group)
		return
	}
}

// finishDrag applies the pending drop, if any, and clears the drag state.
func (d *DockingLayout) finishDrag() {
	if d.dragPanel != nil && d.dropNode != nil && d.dropEdge != dropEdgeNone {
		d.movePanel(d.dragPanel, d.dragGroup, d.dropNode, d.dropEdge)
	}
	d.dragPanel = nil
	d.dragGroup = nil
	d.sourceNode = nil
	d.dropNode = nil
	d.dropEdge = dropEdgeNone
	d.dropRect = image.Rectangle{}
	guigui.RequestRedraw(d)
}

// movePanel removes panel from its source group and re-docks it: into target's
// group on a center drop, or as a new group split next to target on an edge.
func (d *DockingLayout) movePanel(panel *DockPanel, source *DockGroup, targetNode *DockNode, edge DropEdge) {
	if removePanelFromGroup(source, panel) {
		// The source group is now empty; remove it from the tree.
		d.root = removeNode(d.root, findGroupNode(d.root, source))
	}

	if edge == dropEdgeCenter {
		target := targetNode.group
		target.panels = append(target.panels, panel)
		target.selected = len(target.panels) - 1
		return
	}

	newGroup := &DockGroup{
		panels:   []*DockPanel{panel},
		selected: 0,
	}
	d.root = attachNode(d.root, targetNode, &DockNode{group: newGroup}, edge)
}

// removePanelFromGroup removes panel from group, fixing the selection. It
// reports whether the group is now empty.
func removePanelFromGroup(group *DockGroup, panel *DockPanel) bool {
	for i, p := range group.panels {
		if p == panel {
			group.panels = append(group.panels[:i], group.panels[i+1:]...)
			if group.selected >= len(group.panels) {
				group.selected = len(group.panels) - 1
			}
			return len(group.panels) == 0
		}
	}
	return false
}

// findGroupNode returns the node whose group is group, or nil.
func findGroupNode(node *DockNode, group *DockGroup) *DockNode {
	if node == nil {
		return nil
	}
	if node.group == group {
		return node
	}
	if node.split != nil {
		if n := findGroupNode(node.split.first, group); n != nil {
			return n
		}
		if n := findGroupNode(node.split.second, group); n != nil {
			return n
		}
	}
	return nil
}

// removeNode returns node's subtree with the target node removed, collapsing
// the split that directly contained it so the sibling keeps its own identity.
func removeNode(node, target *DockNode) *DockNode {
	if node == target {
		return nil
	}
	if node.split == nil {
		return node
	}
	if node.split.first == target {
		return node.split.second
	}
	if node.split.second == target {
		return node.split.first
	}
	node.split.first = removeNode(node.split.first, target)
	node.split.second = removeNode(node.split.second, target)
	return node
}

// attachNode replaces target with a new split holding target and newNode,
// ordered by edge. Returns the new root.
func attachNode(root, target, newNode *DockNode, edge DropEdge) *DockNode {
	direction, newNodeFirst := splitForEdge(edge)
	newSplit := &DockNode{
		split: &DockSplit{
			direction: direction,
			ratio:     0.5,
		},
	}
	if newNodeFirst {
		newSplit.split.first = newNode
		newSplit.split.second = target
	} else {
		newSplit.split.first = target
		newSplit.split.second = newNode
	}
	if root == target {
		return newSplit
	}
	replaceNode(root, target, newSplit)
	return root
}

// splitForEdge returns the split direction for an edge drop and whether the
// new node should be the split's first child.
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

func replaceNode(node, target, replacement *DockNode) {
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
	replaceNode(node.split.first, target, replacement)
	replaceNode(node.split.second, target, replacement)
}

// dropEdgeAt returns the drop zone the cursor falls in within bounds. Dropping
// on the tab bar (the top strip) tabs the panel into the group; the outer
// thirds are the four edge-split targets; the rest is the center.
func (d *DockingLayout) dropEdgeAt(context *guigui.Context, cursor image.Point, b image.Rectangle) DropEdge {
	u := basicwidget.UnitSize(context)
	if cursor.Y < b.Min.Y+u {
		return dropEdgeCenter
	}
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

func (d *DockingLayout) dropRectFor(context *guigui.Context, cursor image.Point, b image.Rectangle, edge DropEdge, group *DockGroup) image.Rectangle {
	u := basicwidget.UnitSize(context)
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
	case dropEdgeCenter:
		if cursor.Y < b.Min.Y+u {
			// The panel is appended as the last tab, so highlight only the
			// empty part of the tab bar after the existing tabs.
			start := max(b.Min.X, group.tabBarUsed)
			if start >= b.Max.X {
				return image.Rectangle{}
			}
			return image.Rectangle{Min: image.Pt(start, b.Min.Y), Max: image.Pt(b.Max.X, b.Min.Y+u)}
		}
		return image.Rectangle{
			Min: image.Pt(b.Min.X+edgeX, b.Min.Y+edgeY),
			Max: image.Pt(b.Max.X-edgeX, b.Max.Y-edgeY),
		}
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
	if d.dragPanel != nil {
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
