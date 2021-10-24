package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func choice(prompt string, choices []string) (int, error) {
	for i, choice := range choices {
		fmt.Printf("%v) %v\n", i+1, choice)
	}
	fmt.Printf("\n%v [1-%v or exit]", prompt, len(choices))
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("error reading choice")
	}
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "exit" {
		return 0, fmt.Errorf("aborted by user")
	}

	index, err := strconv.Atoi(response)
	if err != nil {
		return 0, fmt.Errorf("invalid input: %v", err)
	}
	if index == 0 || index > len(choices) {
		return 0, fmt.Errorf("invalid input: %v", index)
	}
	return index - 1, nil
}

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
