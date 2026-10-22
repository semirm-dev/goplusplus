package main

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

type Order struct {
	ID       int
	Customer string
	Amount   float64
	Paid     bool
}

func ReportNaive(orders []Order) string {
	var out string
	totals := map[string]float64{}

	for _, o := range orders {
		if !o.Paid {
			continue
		}
		out += fmt.Sprintf("%d,%s,%.2f\n", o.ID, o.Customer, o.Amount)
		totals[o.Customer] += o.Amount
	}

	for _, c := range slices.Sorted(maps.Keys(totals)) {
		out += fmt.Sprintf("total,%s,%.2f\n", c, totals[c])
	}
	return out
}

func ReportTuned(buf []byte, orders []Order) []byte {
	buf = buf[:0]
	totals := make(map[string]float64, 64) // size hint: about as many customers as we expect

	for _, o := range orders {
		if !o.Paid {
			continue
		}
		buf = strconv.AppendInt(buf, int64(o.ID), 10)
		buf = append(buf, ',')
		buf = append(buf, o.Customer...)
		buf = append(buf, ',')
		buf = strconv.AppendFloat(buf, o.Amount, 'f', 2, 64)
		buf = append(buf, '\n')
		totals[o.Customer] += o.Amount
	}

	for _, c := range slices.Sorted(maps.Keys(totals)) {
		buf = append(buf, "total,"...)
		buf = append(buf, c...)
		buf = append(buf, ',')
		buf = strconv.AppendFloat(buf, totals[c], 'f', 2, 64)
		buf = append(buf, '\n')
	}
	return buf
}

func sampleOrders(n int) []Order {
	orders := make([]Order, n)
	for i := range orders {
		orders[i] = Order{
			ID:       i + 1,
			Customer: "customer-" + strconv.Itoa(i%50),
			Amount:   float64(i%997) + 0.99,
			Paid:     i%4 != 0,
		}
	}
	return orders
}
