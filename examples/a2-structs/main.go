// 🌩 Go++ 🌩 #2 · Coding guidelines: Structs
package main

import "fmt"

// Fields directly exported, so you can access them.
// Not: private fields with GetTotal() and SetTotal() that only read and write regular values.
type Order struct {
	ID       string
	Total    int64 // cents
	Currency string
}

// Pass total and currency only, because that's all we need.
// Not: func FormatTotal(o Order) string. No need to pass the entire order struct.
func FormatTotal(total int64, currency string) string {
	return fmt.Sprintf("%d.%02d %s", total/100, total%100, currency)
}

// Passing entire order struct is acceptable, because we use all the fields.
func Save(orders map[string]Order, o Order) {
	orders[o.ID] = o
}

func main() {
	o := Order{ID: "o-1", Total: 1999, Currency: "EUR"}

	orders := map[string]Order{}
	Save(orders, o)

	fmt.Println(FormatTotal(o.Total, o.Currency))
}
