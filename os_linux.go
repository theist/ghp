// +build linux

package main

import "os/exec"

// USER_TOKEN token store filename
const userState = ".ghp.state"

func openBrowser(url string) error {
	err := exec.Command("xdg-open", url).Start()
	return err
}
