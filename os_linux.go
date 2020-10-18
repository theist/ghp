// +build linux

package main

import "os/exec"

// GLOBAL_CONFIG path to config
const GLOBAL_CONFIG = "/etc/ghp.cfg"

// USER_CONFIG path to user config
const USER_CONFIG = ".ghp.cfg"

// USER_TOKEN token store filename
const USER_TOKEN = ".ghp.token"

func openBrowser(url string) error {
	err := exec.Command("xdg-open", url).Start()
	return err
}
