package enforcement

import (
	"bytes"
	"os/exec"
)

type Executor interface {
	Run(cmd string, args ...string) (string, error)
}

type CommandExecutor struct{}

func (c *CommandExecutor) Run(cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}
