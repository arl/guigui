// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Date struct {
	guigui.DefaultWidget

	background basicwidget.Background

	introTitle basicwidget.Text
	introBody  basicwidget.Text

	panel basicwidget.Panel
	form  basicwidget.Form

	dockedLabel  basicwidget.Text
	dockedPicker basicwidget.DockedDatePicker
	dockedValue  basicwidget.Text

	modalLabel  basicwidget.Text
	modalPicker basicwidget.ModalDatePicker
	modalValue  basicwidget.Text

	inputLabel  basicwidget.Text
	inputPicker basicwidget.ModalDateInput
	inputValue  basicwidget.Text

	layoutItems []guigui.LinearLayoutItem
}

func (d *Date) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&d.background)
	adder.AddWidget(&d.introTitle)
	adder.AddWidget(&d.introBody)
	adder.AddWidget(&d.panel)

	d.introTitle.SetValue("Date pickers")
	var titleStyle basicwidget.TextStyle
	titleStyle.SetBold(true)
	d.introTitle.SetBaseStyle(&titleStyle)
	d.introTitle.SetScale(1.5)

	d.introBody.SetValue("Three Material 3 date picker kinds: a docked field with a calendar dropdown, a modal calendar dialog, and a modal text-input dialog. Each calendar uses month and year menus, previous/next month, weekday headings, and Cancel/OK.")
	d.introBody.SetWrapMode(basicwidget.WrapModeNormal)

	d.panel.SetContent(&d.form)
	d.panel.SetAutoBorder(true)
	d.panel.SetContentConstraints(basicwidget.PanelContentConstraintsFixedWidth)

	d.dockedLabel.SetValue("Docked")
	d.dockedPicker.SetPlaceholder("DD/MM/YYYY")
	d.dockedPicker.OnValueChanged(func(context *guigui.Context, date basicwidget.Date) {
		d.dockedValue.SetValue(formatSelected(date))
	})
	d.dockedValue.SetValue(formatSelected(d.dockedPicker.Value()))

	d.modalLabel.SetValue("Modal")
	d.modalPicker.SetTitle("Select date")
	d.modalPicker.OnValueChanged(func(context *guigui.Context, date basicwidget.Date) {
		d.modalValue.SetValue(formatSelected(date))
	})
	d.modalValue.SetValue(formatSelected(d.modalPicker.Value()))

	d.inputLabel.SetValue("Modal input")
	d.inputPicker.SetTitle("Enter date")
	d.inputPicker.OnValueChanged(func(context *guigui.Context, date basicwidget.Date) {
		d.inputValue.SetValue(formatSelected(date))
	})
	d.inputValue.SetValue(formatSelected(d.inputPicker.Value()))

	d.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &d.dockedLabel,
			SecondaryWidget: &d.dockedPicker,
		},
		{
			PrimaryWidget:   nil,
			SecondaryWidget: &d.dockedValue,
		},
		{
			PrimaryWidget:   &d.modalLabel,
			SecondaryWidget: &d.modalPicker,
		},
		{
			PrimaryWidget:   nil,
			SecondaryWidget: &d.modalValue,
		},
		{
			PrimaryWidget:   &d.inputLabel,
			SecondaryWidget: &d.inputPicker,
		},
		{
			PrimaryWidget:   nil,
			SecondaryWidget: &d.inputValue,
		},
	})

	return nil
}

func (d *Date) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&d.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	d.layoutItems = slices.Delete(d.layoutItems, 0, len(d.layoutItems))
	d.layoutItems = append(d.layoutItems,
		guigui.LinearLayoutItem{Widget: &d.introTitle},
		guigui.LinearLayoutItem{Widget: &d.introBody},
		guigui.LinearLayoutItem{
			Widget: &d.panel,
			Size:   guigui.FlexibleSize(1),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     d.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func formatSelected(d basicwidget.Date) string {
	if d.IsZero() {
		return "No date selected"
	}
	return d.Time().Format("Monday, January 2, 2006")
}
