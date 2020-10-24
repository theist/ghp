package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" || response == "s" || response == "si" || response == "sí" {
			return true
		}
		return false
	}
}
