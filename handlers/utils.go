package handlers

import (
	"os"
	"os/exec"
)

type CLIExecutor interface {
	Cd(string) error
	ShellOut(string) (string, error)
}

type Shelly struct {
}

func (s Shelly) Cd(dir string) error {
	return os.Chdir(dir)
}

func (s Shelly) ShellOut(cmd string) (string, error) {
	out, err := exec.Command(cmd).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
