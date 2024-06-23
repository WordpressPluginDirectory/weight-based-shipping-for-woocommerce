package main

import (
	. "e2e/src/lib"
	"fmt"
	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
	"log"
	"math"
	"regexp"
	"strconv"
	"testing"
)

func TestPrimary(t *testing.T) {

	var page playwright.Page
	var wc *WcRemote
	beforeEach := func(t *testing.T) {

		Restore()

		page = StartBrowser(t)
		must(page.Goto("/wp-admin/admin.php?page=wc-settings&tab=shipping&zone_id=1"))

		addShippingMethod(page, "wbs")

		must(0, Expect(page.Locator(".wbs-me-welcome-title")).ToHaveText("Add shipping rules"))
		must(0, page.Locator(".wbs-me-welcome-button").Click())

		wc = NewWcRemote(page)
	}

	t.Run("add rules", func(t *testing.T) {
		beforeEach(t)

		free, regular, express1, express2 := rules()

		all := Rules{free, regular, express1, express2}
		all.TypeIn(page)
		all.CheckSaveReloadCheck(t, page)

		t.Run("delivery options", func(t *testing.T) {

			prevurl := page.URL()
			t.Cleanup(func() { must(page.Goto(prevurl)) })
			startWithEmptyCart(page, wc)

			// 1kg, $19, no class
			const regularProduct = "regular-product"
			const regularProductPrice = 19.0
			const regularProductQty = 1
			wc.AddToCart(regularProduct, regularProductQty)

			requireDeliveryOptions(t, wc,
				"Regular Shipping", "2.60",
				"Express Shipping", "19.95",
			)

			// 5kg, $25, bulky-items class
			bulkyProduct := "bulky-product"
			bulkyProductPrice := 25.0
			wc.AddToCart(bulkyProduct, 1)

			requireDeliveryOptions(t, wc,
				"Regular Shipping", "9.10",
				"Express Shipping", "42.50",
			)

			freeThreshold := must(strconv.ParseFloat(*free.OrderSubtotal.Start, 64))
			regularProductQtyForFreeShipping := int(math.Ceil((freeThreshold - bulkyProductPrice) / regularProductPrice))
			wc.AddToCart(regularProduct, regularProductQtyForFreeShipping-regularProductQty)

			requireDeliveryOptions(t, wc,
				"Free Shipping", "0",
			)
		})

		t.Run("delete all rules", func(t *testing.T) {

			page.OnDialog(func(dialog playwright.Dialog) {
				require.Equal(t, fmt.Sprintf("Are you sure you want to delete %d items?", len(all)), dialog.Message())
				must(0, dialog.Accept())
			})
			must(0, page.Locator(".wbs-rse-checkbox").First().Click())
			must(0, page.Locator(".wbs-rse-action-group").GetByRole("button", Role{Name: "Delete"}).Click())

			var all Rules
			all.CheckSaveReloadCheck(t, page)
		})
	})

	if Meta.Edition == Free {
		return
	}

	t.Run("add a rule with classes", func(t *testing.T) {
		beforeEach(t)

		rule := ruleWithClasses()

		all := Rules{rule}
		all.TypeIn(page)
		all.CheckSaveReloadCheck(t, page)

		t.Run("delivery option with classes", func(t *testing.T) {

			startWithEmptyCart(page, wc)

			option := NewDeliveryOption("Shipping With Classes", "0")

			// 1kg, $145, free-shipping
			wc.AddToCart("free-shipping-product", 10)
			requireDeliveryOptions(t, wc, option)

			// 5kg, $25, bulky-items
			wc.AddToCart("bulky-product", 3)
			option.AddPrice(31 + 20*2)
			requireDeliveryOptions(t, wc, option)

			// 0kg, $10, no class
			wc.AddToCart("zero-weight-product", 10)
			// 1.5kg, $19, no class
			wc.AddToCart("regular-product", 1)
			option.AddPrice(17 + 3.5*2)
			requireDeliveryOptions(t, wc, option)
		})
	})

	t.Run("delivery option with and without classes", func(t *testing.T) {
		beforeEach(t)

		free, regular, express1, express2 := rules()
		withClasses := ruleWithClasses()

		all := Rules{free, regular, express1, express2, withClasses}
		all.TypeIn(page)
		all.CheckSaveReloadCheck(t, page)

		startWithEmptyCart(page, wc)

		// 5kg, $25, bulky-items
		wc.AddToCart("bulky-product", 3)

		// 0kg, $10, no class
		wc.AddToCart("zero-weight-product", 1)

		requireDeliveryOptions(t, wc,
			"Regular Shipping", 1.3*15,
			"Express Shipping", 35+20*2.5,
			"Shipping With Classes", 31+20*2+17,
		)
	})
}

func requireDeliveryOptions(t *testing.T, wc *WcRemote, args ...any) {
	t.Helper()

	var options []DeliveryOption
	for len(args) > 0 {

		var option DeliveryOption
		switch a := args[0].(type) {
		case string:
			option = NewDeliveryOption(a, args[1])
			args = args[2:]
		case DeliveryOption:
			option = a
			args = args[1:]
		default:
			panic("invalid delivery option format")
		}

		options = append(options, option)
	}

	require.Equal(t, options, wc.GetDeliveryOptions())
}

func rules() (free, regular, express1, express2 *Rule) {

	free = &Rule{
		Title:         "Free Shipping",
		OrderSubtotal: Range{}.From("250", true),
	}

	regular = &Rule{
		Title:         "Regular Shipping",
		WeightRate:    &WeightRate{Charge: p("1.3"), ForEach: p("1")},
		OrderSubtotal: Range{}.To("250", false),
	}

	express1 = &Rule{
		Title:         "Express Shipping",
		BaseCost:      p("19.95"),
		OrderSubtotal: regular.OrderSubtotal,
		OrderWeight:   Range{}.To("5", false),
	}

	express2 = &Rule{
		Title:         "Express Shipping",
		BaseCost:      p("35"),
		OrderSubtotal: regular.OrderSubtotal,
		OrderWeight:   Range{}.From("5", false),
		WeightRate:    &WeightRate{p("2.5"), p("0.5"), p("5")},
	}

	return free, regular, express1, express2
}

func ruleWithClasses() *Rule {
	return &Rule{
		Title: "Shipping With Classes",
		Classes: ShClasses{
			&ShClass{
				Name: p("Free Shipping"),
			},
			&ShClass{
				Name:           p("No shipping class"),
				AdditionalCost: p("17"),
				WeightRate: &WeightRate{
					Charge:  p("3.5"),
					ForEach: p("0.8"),
				},
			},
			&ShClass{
				Name:           p("Bulky Items"),
				AdditionalCost: p("31"),
				WeightRate: &WeightRate{
					Charge:  p("20"),
					ForEach: p("3"),
					Over:    p("10"),
				},
			},
		},
	}
}

func startWithEmptyCart(page playwright.Page, wc *WcRemote) {

	wc.GotoCart()

	empty := page.Locator("body", playwright.PageLocatorOptions{HasText: "Your cart is currently empty"})
	remove := page.GetByLabel(regexp.MustCompile("Remove .* from cart")).First()
	must(0, Expect(empty.Or(remove)).ToBeVisible())

	if must(remove.IsVisible()) {
		log.Print("the cart is not empty; clear")
	}
	for must(remove.IsVisible()) {
		must(0, remove.Click())
	}
}

func addShippingMethod(page playwright.Page, id string) {

	must(0, page.GetByRole("button", PRole{Name: "Add shipping method"}).Click())

	modalMenuItem := page.Locator(fmt.Sprintf("label[for=%s]", id))
	selectBox := page.Locator("select[name=add_method_id]")
	must(0, modalMenuItem.Or(selectBox).WaitFor())
	if must(modalMenuItem.IsVisible()) {
		// WC 8.4+
		must(0, modalMenuItem.Click())
		must(0, page.GetByRole("button", PRole{Name: "Continue"}).Click())
		must(0, page.Locator(".wc-shipping-zone-action-edit").Last().Click())
	} else {
		// WC < 8.4
		must(selectBox.SelectOption(playwright.SelectOptionValues{ValuesOrLabels: &[]string{id}}))
		must(0, page.Locator("#btn-ok").Click())
		must(0, page.Locator("[href*=instance_id]").Last().Click())
	}
}
