// 🌩 Go++ 🌩 #1 · Coding guidelines: Naming
//
// Package names: transport, discount, inventory.
// Not: transport_utils, transportUtils, services, controllers.
package main

import "fmt"

// CamelCase, even for constants.
// Not: MAX_RETRIES
const maxRetries = 3

// No stuttering, it reads well -> transport.Client.
// Not: TransportClient which is then called like transport.TransportClient.
//
// No need for suffixes like ClientImpl, BaseClient, DefaultClient.
// It is either a specific implementation or it is not an implementation at all.
type Client struct {
	BaseURL string
}

func (c Client) Send(paths []string) {
	// Short names are fine, the declaration is right here.
	// Not: for currentPathIndex, currentPath := range paths
	for i, p := range paths {
		fmt.Printf("%d: %s%s (retries %d)\n", i, c.BaseURL, p, maxRetries)
	}
}

func main() {
	c := Client{BaseURL: "https://api.example.com"}
	c.Send([]string{"/orders", "/users"})
}
