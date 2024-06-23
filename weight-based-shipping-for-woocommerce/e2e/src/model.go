package main

import (
	"e2e/src/lib"
	"fmt"
	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
	"testing"
)

type Rules []*Rule

type Rule struct {
	Title         string
	OrderWeight   *Range
	OrderSubtotal *Range
	BaseCost      *string
	WeightRate    *WeightRate
	Classes       ShClasses
}

type Range struct {
	Start, End           *string
	StartEqual, EndEqual *bool
}

type WeightRate struct {
	Charge, ForEach, Over *string
}

type ShClasses []*ShClass

type ShClass struct {
	Name           *string
	AdditionalCost *string
	WeightRate     *WeightRate
}

func (r Rules) TypeIn(page playwright.Page) {
	for _, rule := range r {
		rule.TypeIn(page)
	}
}

func (r Rules) Save(page playwright.Page) {
	must(0, page.Locator("save-button button").Click())
	must(0, Expect(page.Locator("body")).ToContainText("Your settings have been saved"))
}

func (r Rules) Check(t *testing.T, page playwright.Page) {

	var got []string
	rows := must(page.Locator(".wbs-rse-rule").All())
	for _, row := range rows {
		got = append(got, must(row.Locator("td").Nth(-1-4).TextContent()))
	}

	var want []string
	for _, rule := range r {
		want = append(want, rule.Title)
	}

	require.Equal(t, want, got)
}

func (r Rules) CheckSaveReloadCheck(t *testing.T, page playwright.Page) {

	r.Check(t, page)

	r.Save(page)
	r.Check(t, page)

	must(page.Reload())
	r.Check(t, page)
}

func (r *Rule) TypeIn(page playwright.Page) {

	must(0, page.GetByRole("button", lib.PRole{Name: "Add new"}).Click())
	defer func() {
		must(0, page.Keyboard().Press("Escape"))
	}()

	must(0, page.Locator(wcformrow("Title")).Locator("input").Fill(r.Title))
	if r.OrderWeight != nil {
		r.OrderWeight.TypeIn(page, "Order Weight")
	}
	if r.OrderSubtotal != nil {
		r.OrderSubtotal.TypeIn(page, "Order Subtotal")
	}
	if r.BaseCost != nil {
		must(0, page.Locator(wcformrow("Base Cost")).Locator("input").Fill(*r.BaseCost))
	}
	if r.WeightRate != nil {
		r.WeightRate.TypeIn(page.Locator(wcformrow("Weight Rate")))
	}
	r.Classes.TypeIn(page)
}

func (r Range) From(start string, equal bool) *Range {
	res := r
	res.Start = &start
	res.StartEqual = &equal
	return &res
}

func (r Range) To(end string, equal bool) *Range {
	res := r
	res.End = &end
	res.EndEqual = &equal
	return &res
}

func (r *Range) TypeIn(page playwright.Page, field string) {
	root := page.Locator(wcformrow(field))
	r.typeInBound(root.Locator("range-bound-editor[label=above]"), r.Start, r.StartEqual)
	r.typeInBound(root.Locator("range-bound-editor[label=below]"), r.End, r.EndEqual)
}

func (r *Range) typeInBound(root playwright.Locator, value *string, equal *bool) {
	if value != nil {
		must(0, root.Locator("input[type=text]").Fill(*value))
	}
	if equal != nil {
		must(0, root.Locator("input[type=checkbox]").SetChecked(*equal))
	}
}

func (r *WeightRate) TypeIn(root playwright.Locator) {
	r.typeInField(root, "charge", r.Charge)
	r.typeInField(root, "for each", r.ForEach)
	r.typeInField(root, "over", r.Over)
}

func (r *WeightRate) typeInField(root playwright.Locator, field string, value *string) {
	if value != nil {
		must(0, root.Locator("weight-charge-editor").
			Locator("label", lib.LocOpts{HasText: field}).
			Locator("input").
			Fill(*value))
	}
}

func (c ShClasses) TypeIn(page playwright.Page) {
	root := page.Locator("shipping-class-rates-editor")
	for _, class := range c {
		must(0, root.Locator(".wbs-scr-add-button").Click())
		row := root.Locator(".wbs-scr-table-row").Last()
		if class.Name != nil {
			must(row.Locator("select").SelectOption(lib.SelectOpts{Labels: &[]string{*class.Name}}))
		}
		if class.AdditionalCost != nil {
			must(0, row.Locator("base-charge-editor input").Fill(*class.AdditionalCost))
		}
		if class.WeightRate != nil {
			class.WeightRate.TypeIn(row)
		}
	}
}

func wcformrow(field string) string {
	return fmt.Sprintf(`[wcformrow="%s"]`, field)
}

func p[T any](v T) *T {
	return &v
}
