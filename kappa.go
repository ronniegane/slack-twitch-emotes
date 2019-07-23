package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	var team, email, password string
	flag.StringVar(&team, "team", "", "your team or workspace name")
	flag.StringVar(&email, "email", "", "the email address you use for this slack team")
	flag.StringVar(&password, "password", "", "your password for this slack team")
	flag.Parse()

	// Team and email address are required
	if len(team) == 0 || len(email) == 0 {
		fmt.Println("Team name and email address are required")
		os.Exit(1)
	}

	// If password is missing then ask for it
	for len(password) == 0 {
		fmt.Printf("Password for %s in %s: ", email, team)
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Printf("Failed to read password: %v", err)
		}
		password = string(bytePassword)
		fmt.Println()
	}

	fmt.Printf("team: %q, email: %q, password: %q\n", team, email, password)
}
