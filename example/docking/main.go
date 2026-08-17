// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// Root hosts the docking layout that the example showcases.
type Root struct {
	guigui.DefaultWidget

	background basicwidget.Background
	dock       DockingLayout

	editor  editorPanel
	form    formPanel
	console consolePanel
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.dock)
	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	b := widgetBounds.Bounds()
	layouter.LayoutWidget(&r.background, b)
	layouter.LayoutWidget(&r.dock, b)
}

// editorPanel hosts a multiline rich-text editor.
type editorPanel struct {
	guigui.DefaultWidget

	editor      basicwidget.TextInput
	initialized bool
}

func (e *editorPanel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&e.editor)
	e.editor.SetMultiline(true)
	e.editor.SetWrapMode(basicwidget.WrapModeNormal)
	e.editor.SetLineHeightMode(basicwidget.LineHeightModeFlexible)
	e.editor.SetRichTextEditable(true)
	e.editor.SetPlaceholder("Start writing…")

	// Seed the sample content once, so a later rebuild does not overwrite the
	// user's edits.
	if !e.initialized {
		e.editor.SetValue("A dockable rich-text editor.\n\nSelect text to apply inline styles.")
		e.initialized = true
	}
	return nil
}

func (e *editorPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	layouter.LayoutWidget(&e.editor, widgetBounds.Bounds().Inset(u/2))
}

// formPanel hosts a form with a variety of controls. Its control state lives
// in the struct fields below, updated by the controls' callbacks, so a rebuild
// restores the user's choices rather than resetting them.
type formPanel struct {
	guigui.DefaultWidget

	panel basicwidget.Panel
	form  basicwidget.Form

	nameText  basicwidget.Text
	nameInput basicwidget.TextInput

	priorityGroup basicwidget.RadioButtonGroup[string]
	priorityTexts [3]basicwidget.Text

	enabledText  basicwidget.Text
	enabledCheck basicwidget.Checkbox

	themeText   basicwidget.Text
	themeSelect basicwidget.Select[string]

	opacityText  basicwidget.Text
	opacitySlide basicwidget.Slider

	volumeText  basicwidget.Text
	volumeInput basicwidget.NumberInput

	submitButton basicwidget.Button
	cancelButton basicwidget.Button

	priority string
	enabled  bool
	theme    string
	opacity  int
	volume   int

	initialized bool
}

func (f *formPanel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&f.panel)
	adder.AddWidget(&f.priorityGroup)

	f.panel.SetContent(&f.form)
	f.panel.SetContentConstraints(basicwidget.PanelContentConstraintsFixedWidth)

	if !f.initialized {
		f.priority = "Medium"
		f.enabled = true
		f.theme = "Light"
		f.opacity = 75
		f.volume = 50
		f.initialized = true
	}

	f.nameText.SetValue("Name")
	f.nameInput.SetPlaceholder("Enter a name")

	f.priorityGroup.SetValues([]string{"Low", "Medium", "High"})
	f.priorityTexts[0].SetValue("Low")
	f.priorityTexts[1].SetValue("Medium")
	f.priorityTexts[2].SetValue("High")
	f.priorityGroup.SelectItemByValue(f.priority)
	f.priorityGroup.OnItemSelected(func(context *guigui.Context, index int) {
		if value, ok := f.priorityGroup.SelectedValue(); ok {
			f.priority = value
		}
	})

	f.enabledText.SetValue("Enabled")
	f.enabledCheck.SetValue(f.enabled)
	f.enabledCheck.OnValueChanged(func(context *guigui.Context, value bool) {
		f.enabled = value
	})

	f.themeText.SetValue("Theme")
	f.themeSelect.SetItemsByStrings([]string{"Light", "Dark", "System"})
	f.themeSelect.SelectItemByValue(f.theme)
	f.themeSelect.OnItemSelected(func(context *guigui.Context, index int) {
		if item, ok := f.themeSelect.SelectedItem(); ok {
			f.theme = item.Value
		}
	})

	f.opacityText.SetValue("Opacity")
	f.opacitySlide.SetMinimumValue(0)
	f.opacitySlide.SetMaximumValue(100)
	f.opacitySlide.SetValue(f.opacity)
	f.opacitySlide.OnValueChanged(func(context *guigui.Context, value int) {
		f.opacity = value
	})

	f.volumeText.SetValue("Volume")
	f.volumeInput.SetMinimumValue(0)
	f.volumeInput.SetMaximumValue(100)
	f.volumeInput.SetValue(f.volume)
	f.volumeInput.OnValueChanged(func(context *guigui.Context, value int, committed bool) {
		f.volume = value
	})

	f.submitButton.SetText("Submit")
	f.cancelButton.SetText("Cancel")

	f.form.SetItems([]basicwidget.FormItem{
		{PrimaryWidget: &f.nameText, SecondaryWidget: &f.nameInput},
		{PrimaryWidget: &f.priorityTexts[0], SecondaryWidget: f.priorityGroup.RadioButton(0)},
		{PrimaryWidget: &f.priorityTexts[1], SecondaryWidget: f.priorityGroup.RadioButton(1)},
		{PrimaryWidget: &f.priorityTexts[2], SecondaryWidget: f.priorityGroup.RadioButton(2)},
		{PrimaryWidget: &f.enabledText, SecondaryWidget: &f.enabledCheck},
		{PrimaryWidget: &f.themeText, SecondaryWidget: &f.themeSelect},
		{PrimaryWidget: &f.opacityText, SecondaryWidget: &f.opacitySlide},
		{PrimaryWidget: &f.volumeText, SecondaryWidget: &f.volumeInput},
		{SecondaryWidget: &f.submitButton},
		{SecondaryWidget: &f.cancelButton},
	})
	return nil
}

func (f *formPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&f.panel, widgetBounds.Bounds())
}

// consolePanel hosts a scratchpad that demonstrates a third docked panel.
type consolePanel struct {
	guigui.DefaultWidget

	console basicwidget.TextInput
}

func (c *consolePanel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.console)
	c.console.SetMultiline(true)
	c.console.SetWrapMode(basicwidget.WrapModeNormal)
	c.console.SetPlaceholder("Type a note…")
	return nil
}

func (c *consolePanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	layouter.LayoutWidget(&c.console, widgetBounds.Bounds().Inset(u/2))
}

func main() {
	root := &Root{}

	// Nest a horizontal split (form | editor) inside a vertical split with a
	// console panel docked at the bottom.
	root.dock.SetRoot(&DockNode{
		split: &DockSplit{
			direction: guigui.LayoutDirectionVertical,
			ratio:     0.7,
			first: &DockNode{
				split: &DockSplit{
					direction: guigui.LayoutDirectionHorizontal,
					ratio:     0.35,
					first:     &DockNode{panel: &DockPanel{title: "Form", content: &root.form, Pinned: true}},
					second:    &DockNode{panel: &DockPanel{title: "Editor", content: &root.editor, Pinned: true}},
				},
			},
			second: &DockNode{panel: &DockPanel{title: "Console", content: &root.console, Pinned: true}},
		},
	})

	if err := guigui.Run(root, &guigui.RunOptions{
		Title:      "Docking",
		WindowSize: image.Pt(960, 640),
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
