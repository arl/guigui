// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// DockGroup is a leaf in the docking tree that holds one or more panels shown
// as tabs. The selected panel's content is displayed below the tab bar.
type DockGroup struct {
	guigui.DefaultWidget

	panels   []*DockPanel
	selected int

	// onDragStart is wired by the DockingLayout to begin moving a panel whose
	// tab was pressed.
	onDragStart func(panel *DockPanel, cursor image.Point)

	// onGroupDragStart is wired by the DockingLayout to begin moving the whole
	// group, called when the empty part of the tab bar is pressed.
	onGroupDragStart func(group *DockGroup, cursor image.Point)

	// onTabClick, when set, overrides pressTab for tab presses. Used by edge
	// bars to implement expand/collapse/select instead of a drag.
	onTabClick func(panel *DockPanel, cursor image.Point)

	// vertical renders the tab strip on a side instead of across the top.
	vertical     bool
	stripOnRight bool
	// collapsed hides the content when vertical (the bar shows only the strip).
	collapsed bool

	tabs guigui.WidgetSlice[*groupTab]

	layoutItems []guigui.LinearLayoutItem

	// tabBarUsed is the absolute X where the tabs end and the empty part of
	// the tab bar begins. It is set during Layout (horizontal groups).
	tabBarUsed int
	// tabBarUsedY is the absolute Y where the tabs end and the empty part of
	// the vertical strip begins. It is set during Layout (vertical groups).
	tabBarUsedY int
}

// stripWidth is the width of a vertical tab strip.
func stripWidth(u int) int { return u * 2 }

// pressTab selects panel and starts a drag of it.
func (g *DockGroup) pressTab(panel *DockPanel, cursor image.Point) {
	for i, p := range g.panels {
		if p == panel {
			g.selected = i
			break
		}
	}
	if g.onDragStart != nil {
		g.onDragStart(panel, cursor)
	}
}

// HandlePointingInput starts a whole-group drag when the empty part of the tab
// bar (right of the tabs) is pressed. Vertical edge bars cannot be group-dragged.
func (g *DockGroup) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if g.vertical || !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return guigui.HandleInputResult{}
	}
	cursor := image.Pt(ebiten.CursorPosition())
	b := widgetBounds.Bounds()
	u := basicwidget.UnitSize(context)
	if cursor.Y >= b.Min.Y && cursor.Y < b.Min.Y+u && cursor.X >= g.tabBarUsed {
		if g.onGroupDragStart != nil {
			g.onGroupDragStart(g, cursor)
			return guigui.HandleInputByWidget(g)
		}
	}
	return guigui.HandleInputResult{}
}

func (g *DockGroup) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	g.tabs.SetLen(len(g.panels))
	for i, panel := range g.panels {
		tab := g.tabs.At(i)
		tab.group = g
		tab.panel = panel
		tab.active = i == g.selected
		adder.AddWidget(tab)
	}
	if !g.collapsed && g.selected >= 0 && g.selected < len(g.panels) {
		adder.AddWidget(g.panels[g.selected].Content)
	}
	return nil
}

func (g *DockGroup) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	b := widgetBounds.Bounds()

	if g.vertical {
		sw := stripWidth(u)
		var strip, content image.Rectangle
		if g.stripOnRight {
			strip = image.Rectangle{Min: image.Pt(b.Max.X-sw, b.Min.Y), Max: b.Max}
			content = image.Rectangle{Min: b.Min, Max: image.Pt(b.Max.X-sw, b.Max.Y)}
		} else {
			strip = image.Rectangle{Min: b.Min, Max: image.Pt(b.Min.X+sw, b.Max.Y)}
			content = image.Rectangle{Min: image.Pt(b.Min.X+sw, b.Min.Y), Max: b.Max}
		}

		g.layoutItems = slices.Delete(g.layoutItems, 0, len(g.layoutItems))
		for i := range g.tabs.Len() {
			g.layoutItems = append(g.layoutItems, guigui.LinearLayoutItem{Widget: g.tabs.At(i)})
		}
		vertical := guigui.LinearLayout{Direction: guigui.LayoutDirectionVertical, Items: g.layoutItems}
		// Record where the tabs end so the empty remainder of the strip can be
		// used as a drop-preview target.
		var itemBounds []image.Rectangle
		itemBounds = vertical.AppendItemBounds(itemBounds, context, strip)
		g.tabBarUsedY = strip.Min.Y
		for _, ib := range itemBounds {
			if ib.Max.Y > g.tabBarUsedY {
				g.tabBarUsedY = ib.Max.Y
			}
		}
		vertical.LayoutWidgets(context, strip, layouter)

		if !g.collapsed && g.selected >= 0 && g.selected < len(g.panels) {
			layouter.LayoutWidget(g.panels[g.selected].Content, content)
		}
		return
	}

	tabBar := image.Rectangle{
		Min: b.Min,
		Max: image.Pt(b.Max.X, b.Min.Y+u),
	}

	g.layoutItems = slices.Delete(g.layoutItems, 0, len(g.layoutItems))
	for i := range g.tabs.Len() {
		g.layoutItems = append(g.layoutItems, guigui.LinearLayoutItem{Widget: g.tabs.At(i)})
	}
	linear := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     g.layoutItems,
	}
	// Record where the tabs end so the empty remainder of the tab bar can be
	// used as a drop-preview target.
	var itemBounds []image.Rectangle
	itemBounds = linear.AppendItemBounds(itemBounds, context, tabBar)
	g.tabBarUsed = tabBar.Min.X
	for _, ib := range itemBounds {
		if ib.Max.X > g.tabBarUsed {
			g.tabBarUsed = ib.Max.X
		}
	}
	linear.LayoutWidgets(context, tabBar, layouter)

	if g.selected >= 0 && g.selected < len(g.panels) {
		layouter.LayoutWidget(g.panels[g.selected].Content, image.Rectangle{
			Min: image.Pt(b.Min.X, tabBar.Max.Y),
			Max: b.Max,
		})
	}
}

func (g *DockGroup) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	u := basicwidget.UnitSize(context)
	b := widgetBounds.Bounds()
	var clr color.RGBA
	if context.ColorMode() == ebiten.ColorModeLight {
		clr = color.RGBA{0xc0, 0xc0, 0xc0, 0xff}
	} else {
		clr = color.RGBA{0x38, 0x38, 0x38, 0xff}
	}
	if g.vertical {
		// Separator between the vertical strip and the content.
		sw := stripWidth(u)
		if g.stripOnRight {
			vector.StrokeLine(dst, float32(b.Max.X-sw), float32(b.Min.Y), float32(b.Max.X-sw), float32(b.Max.Y), 1, clr, false)
		} else {
			vector.StrokeLine(dst, float32(b.Min.X+sw), float32(b.Min.Y), float32(b.Min.X+sw), float32(b.Max.Y), 1, clr, false)
		}
		return
	}
	// A separator line under the tab bar.
	vector.StrokeLine(dst, float32(b.Min.X), float32(b.Min.Y+u), float32(b.Max.X), float32(b.Min.Y+u), 1, clr, false)
}

// groupTab is the clickable, draggable label for one panel in a group's tab bar.
type groupTab struct {
	guigui.DefaultWidget

	group  *DockGroup
	panel  *DockPanel
	active bool

	label basicwidget.Text
}

func (t *groupTab) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if !t.group.vertical {
		adder.AddWidget(&t.label)
		t.label.SetValue(t.panel.Title)
	}
	var style basicwidget.TextStyle
	if t.active {
		style.SetBold(true)
	}
	t.label.SetBaseStyle(&style)
	t.label.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	t.label.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	return nil
}

func (t *groupTab) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && widgetBounds.IsHitAtCursor() {
		cursor := image.Pt(ebiten.CursorPosition())
		if t.group.onTabClick != nil {
			t.group.onTabClick(t.panel, cursor)
		} else {
			t.group.pressTab(t.panel, cursor)
		}
		return guigui.HandleInputByWidget(t)
	}
	return guigui.HandleInputResult{}
}

func (t *groupTab) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	layouter.LayoutWidget(&t.label, widgetBounds.Bounds().Inset(u/4))
}

func (t *groupTab) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	if t.group.vertical {
		face := t.face(context)
		width, height := text.Measure(t.panel.Title, face, 0)
		u := basicwidget.UnitSize(context)
		return image.Pt(int(math.Ceil(height))+u/2, int(math.Ceil(width))+u/2)
	}
	u := basicwidget.UnitSize(context)
	s := t.label.Measure(context, constraints)
	return image.Pt(s.X+u/2, u)
}

func (t *groupTab) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return ebiten.CursorShapePointer, true
}

func (t *groupTab) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	b := widgetBounds.Bounds()
	var clr color.RGBA
	if t.active {
		if context.ColorMode() == ebiten.ColorModeLight {
			clr = color.RGBA{0xff, 0xff, 0xff, 0xff}
		} else {
			clr = color.RGBA{0x30, 0x30, 0x30, 0xff}
		}
	} else {
		if context.ColorMode() == ebiten.ColorModeLight {
			clr = color.RGBA{0xe2, 0xe2, 0xe2, 0xff}
		} else {
			clr = color.RGBA{0x22, 0x22, 0x22, 0xff}
		}
	}
	vector.DrawFilledRect(dst, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), clr, false)
	if !t.group.vertical {
		return
	}

	face := t.face(context)
	width, height := text.Measure(t.panel.Title, face, 0)
	var op text.DrawOptions
	op.ColorScale.ScaleWithColor(t.textColor(context))
	if t.group.stripOnRight {
		op.GeoM.Rotate(math.Pi / 2)
		op.GeoM.Translate(float64(b.Max.X)-float64(b.Dx()-int(math.Ceil(height)))/2, float64(b.Min.Y)+(float64(b.Dy())-width)/2)
	} else {
		op.GeoM.Rotate(-math.Pi / 2)
		op.GeoM.Translate(float64(b.Min.X)+(float64(b.Dx())-height)/2, float64(b.Max.Y)-(float64(b.Dy())-width)/2)
	}
	text.Draw(dst, t.panel.Title, face, &op)
}

func (t *groupTab) face(context *guigui.Context) *text.GoTextFace {
	face := &text.GoTextFace{
		Source: basicwidget.DefaultFaceSourceEntry().FaceSource,
		Size:   basicwidget.FontSize(context),
	}
	if t.active {
		face.SetVariation(text.MustParseTag("wght"), 700)
	}
	return face
}

func (t *groupTab) textColor(context *guigui.Context) color.RGBA {
	if context.ColorMode() == ebiten.ColorModeLight {
		return color.RGBA{0x20, 0x20, 0x20, 0xff}
	}
	return color.RGBA{0xe0, 0xe0, 0xe0, 0xff}
}
