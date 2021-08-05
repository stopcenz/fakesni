package main

import (
    "fmt"
    "log"
    "os/exec"
    "runtime"
    "time"
)


func runBrowser(u string, prompt string) {
	time.Sleep(1 * time.Second)
	fmt.Println(prompt)
	// https://dwheeler.com/essays/open-files-urls.html
	var cmd string
	var arg string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		arg = "/c start " + u
	case "darwin":
		cmd = "open"
		arg = u
	case "linux":
		cmd = "xdg-open"
		arg = u
	default:
		return
	}
	err := exec.Command(cmd, arg).Start()
	if err != nil {
		log.Print("Error " + err.Error())
	}
}
