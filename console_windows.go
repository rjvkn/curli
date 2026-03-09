package main

import "golang.org/x/sys/windows"

func setupWindowsConsole(stdoutFd int) error {
	console := windows.Handle(stdoutFd)
	var originalMode uint32
	if err := windows.GetConsoleMode(console, &originalMode); err != nil {
		return err
	}
	if err := windows.SetConsoleOutputCP(windows.CP_UTF8); err != nil {
		return err
	}
	if err := windows.SetConsoleCP(windows.CP_UTF8); err != nil {
		return err
	}
	return windows.SetConsoleMode(console, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
