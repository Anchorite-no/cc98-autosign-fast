//go:build !windows

package main

func initConsole() {}

func shouldPauseOnExit() bool {
	return false
}

func waitForExit() {}
