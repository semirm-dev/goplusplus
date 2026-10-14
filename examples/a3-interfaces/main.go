// 🌩 Go++ 🌩 #3 · Coding guidelines: Interfaces
package main

import "fmt"

type PaymentClient struct {
	charger Charger
}

// Defined where it's used.
type Charger interface {
	Charge(orderID string, amount int64) error
}

// Accept interface, return struct.
func NewPaymentClient(charger Charger) *PaymentClient {
	return &PaymentClient{charger: charger}
}

func (p *PaymentClient) Pay(orderID string, amount int64) error {
	return p.charger.Charge(orderID, amount)
}

// In some other package (payments/stripe.go):
// Implements Charger interface + adds extra methods.
// Now callers of StripeAPI can use both Charge and Refund, while the PaymentClient stays clean.
type StripeAPI struct{}

func (StripeAPI) Charge(orderID string, amount int64) error {
	fmt.Println("charged", orderID, amount)
	return nil
}

func (StripeAPI) Refund(orderID string, amount int64) error {
	fmt.Println("refunded", orderID, amount)
	return nil
}

// In tests file. Fake implementation of Charger interface.
type fakeCharger struct{}

func (fakeCharger) Charge(string, int64) error { return nil }

func main() {
	_ = NewPaymentClient(StripeAPI{}).Pay("o-1", 1999)
	_ = NewPaymentClient(fakeCharger{}).Pay("o-1", 1999)
}
