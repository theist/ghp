// +build linux

package main

import "os/exec"

// globalConfig path to config
const globalConfig = "/etc/ghp.cfg"

// userConfig path to user config
const userConfig = ".ghp.cfg"

// USER_TOKEN token store filename
const userState = ".ghp.state"

func openBrowser(url string) error {
	err := exec.Command("xdg-open", url).Start()
	return err
}
