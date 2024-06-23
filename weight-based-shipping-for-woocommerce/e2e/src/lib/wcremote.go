package lib

import (
	"fmt"
	"github.com/playwright-community/playwright-go"
	"net/url"
	"strconv"
	"strings"
)

type WcRemote struct {
	page playwright.Page
}

func NewWcRemote(page playwright.Page) *WcRemote {
	return &WcRemote{page}
}

func (r *WcRemote) AddToCart(productSlug string, qty int) {

	if qty == 0 {
		return
	}
	if qty < 0 {
		panic("qty must be greater or equal to 0")
	}

	defer func(url string) { must(r.page.Goto(url)) }(r.page.URL())

	must(r.page.Goto(productUrl(productSlug)))
	must(0, r.page.Locator(`//input[starts-with(@id,"quantity_") or @name="quantity"]`).Fill(strconv.Itoa(qty)))
	must(0, r.page.Locator(`//button[@name="add-to-cart"]`).Click())
}

func (r *WcRemote) GetDeliveryOptions() []DeliveryOption {

	if !strings.HasPrefix(r.page.URL(), "/cart") {
		defer func(url string) { must(r.page.Goto(url)) }(r.page.URL())
		r.GotoCart()
	}

	var res []DeliveryOption
	options := must(r.page.Locator("ul#shipping_method li").All())
	for _, option := range options {

		parts := strings.Split(must(option.Locator("label").InnerText()), ":")
		if len(parts) == 1 {
			parts = append(parts, "0")
		}

		parts[1] = strings.TrimLeft(strings.TrimSpace(parts[1]), "£$")

		res = append(res, NewDeliveryOption(parts[0], parts[1]))
	}

	return res
}

func (r *WcRemote) GotoCart() {
	must(r.page.Goto("/cart"))
}

type DeliveryOption struct {
	Title, Price string
}

func NewDeliveryOption(title string, price any) DeliveryOption {

	var sprice string
	switch p := price.(type) {
	case float64:
		sprice = strconv.FormatFloat(p, 'f', -1, 64)
	case int:
		sprice = strconv.Itoa(p)
	case string:
		sprice = p
	default:
		panic("invalid price type")
	}

	return DeliveryOption{title, normalizePrice(sprice)}
}

func (do *DeliveryOption) AddPrice(delta float64) {
	p := must(strconv.ParseFloat(do.Price, 64))
	p += delta
	do.Price = normalizePrice(strconv.FormatFloat(p, 'f', -1, 64))
}

func normalizePrice(price string) string {

	if price == "" {
		return ""
	}

	parts := strings.Split(price, ".")
	if len(parts) == 1 {
		parts = append(parts, "")
	}
	if len(parts) == 2 {
		parts[1] = padRight(parts[1], 2, '0')
		price = strings.Join(parts, ".")
	}

	return price
}

func productUrl(productSlug string) string {
	return fmt.Sprintf("/product/%s/", url.PathEscape(productSlug))
}

func padRight(s string, n int, c rune) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(string(c), n-len(s))
}
