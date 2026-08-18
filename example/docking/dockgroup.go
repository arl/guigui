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

	tabs guigui.WidgetSlice[*groupTab]

	layoutItems []guigui.LinearLayoutItem

	// tabBarUsed is the absolute X where the tabs end and the empty part of
	// the tab bar begins. It is set during Layout.
	tabBarUsed int
}

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
// bar (right of the tabs) is pressed.
func (g *DockGroup) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
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
	if g.selected >= 0 && g.selected < len(g.panels) {
		adder.AddWidget(g.panels[g.selected].Content)
	}
	return nil
}

func (g *DockGroup) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	b := widgetBounds.Bounds()
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
	adder.AddWidget(&t.label)
	t.label.SetValue(t.panel.Title)
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
		t.group.pressTab(t.panel, image.Pt(ebiten.CursorPosition()))
		return guigui.HandleInputByWidget(t)
	}
	return guigui.HandleInputResult{}
}

func (t *groupTab) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	layouter.LayoutWidget(&t.label, widgetBounds.Bounds().Inset(u/4))
}

func (t *groupTab) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
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
}
