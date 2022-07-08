package util

import (
	"bytes"
	"errors"
	"os/exec"

	"github.com/sirupsen/logrus"
)

func RunLinuxCommand(command string) (string, string, error) {
	var outBuff, errBuff bytes.Buffer
	cmd := exec.Command("/bin/bash", "-c", command)
	cmd.Stdout, cmd.Stderr = &outBuff, &errBuff

	defer func() {
		logrus.Infof("Run command: '%s' \n "+
			"stdout: %s\n stderr: %s\n", command, outBuff.String(), errBuff.String())
	}()

	//Run cmd
	if err := cmd.Start(); err != nil {
		logrus.Warningf("Exec command: %s, error: %v", command, err)
		return "", "", err
	}

	//Wait cmd run finish
	if err := cmd.Wait(); err != nil {
		logrus.Warningf("Wait command: %s exec finish error: %v", command, err)
		return "", "", err
	}

	return outBuff.String(), errBuff.String(), nil
}

func RunLinuxCommands(returnStderr bool, commands ...string) error {
	for _, cmd := range commands {
		_, stderr, err := RunLinuxCommand(cmd)
		if err != nil {
			return err
		}
		if stderr != "" && returnStderr {
			return errors.New(stderr)
		}
	}
	return nil
}

func RunLinuxShellFile(filename string) (string, string, error) {
	logrus.Infof("Run shell script %s", filename)

	cmd := exec.Command(filename)
	var outBuff, errBuff bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outBuff, &errBuff

	defer func() {
		logrus.Infof("Run shell script %s \n"+
			"stdout: %s\n stderr: %s\n", filename, outBuff.String(), errBuff.String())
	}()

	err := cmd.Run()
	if err != nil {
		return "", "", err
	}
	return outBuff.String(), errBuff.String(), nil
}
