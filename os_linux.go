// +build linux

package main

import (
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

// USER_TOKEN token store filename
const userState = ".ghp.state"

func openBrowser(url string) error {
	err := exec.Command("xdg-open", url).Start()
	return err
}

func consoleWidth() int {
	ws, _ := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
	if int(ws.Col) > 20 {
		return int(ws.Col)
	}
	return 150
}
