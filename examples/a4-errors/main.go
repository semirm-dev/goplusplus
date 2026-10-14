// 🌩 Go++ 🌩 #4 · Coding guidelines: Error handling
package main

import (
	"errors"
	"fmt"
	"log"
)

// Not: errors.New("User not found.")
var ErrNotFound = errors.New("user not found")

func findUser(id string) error {
	return fmt.Errorf("query user %s: %w", id, ErrNotFound)
}

func fetchUser(id string) error {
	if err := findUser(id); err != nil {
		// Not: log it here and return it as well.
		return fmt.Errorf("user fetch failed: %w", err)
	}
	return nil
}

func main() {
	// Handled once, here.
	if err := fetchUser("42"); errors.Is(err, ErrNotFound) {
		log.Println(err)
	}
}
