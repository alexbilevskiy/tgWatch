package libs

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
)

func Repunc(plaintext string) (string, error) {
	cmd := exec.Command("python3", "cmd/repunc.py")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", errors.New(fmt.Sprintf("failed to execute repunc: %s", err.Error()))
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, plaintext+"\n")
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(fmt.Sprintf("failed to get output of repunc: %s", err.Error()))
	}

	return string(out), nil
}
