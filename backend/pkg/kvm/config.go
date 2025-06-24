package kvm

import (
	"backend/pkg/parse"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func ConfigImage(report *parse.CrashReport) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	scriptDir := filepath.Join(workDir, "script")

	logger := log.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	commitID := report.Crashes[0].KernelSourceCommit
	imageCmd := exec.Command(filepath.Join(scriptDir, "mount.sh"), commitID)
	imageCmd.Stdout = io.MultiWriter(stdout, logger.Writer())
	imageCmd.Stderr = io.MultiWriter(stderr, logger.Writer())
	if err := imageCmd.Run(); err != nil {
		return err
	}

	return nil
}

func ClearImage(report *parse.CrashReport) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	scriptDir := filepath.Join(workDir, "script")

	logger := log.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	commitID := report.Crashes[0].KernelSourceCommit
	clearCmd := exec.Command(filepath.Join(scriptDir, "clear.sh"), commitID)
	clearCmd.Stdout = io.MultiWriter(stdout, logger.Writer())
	clearCmd.Stderr = io.MultiWriter(stderr, logger.Writer())
	if err := clearCmd.Run(); err != nil {
		return err
	}

	return nil
}

func GetVmcore(report *parse.CrashReport) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	scriptDir := filepath.Join(workDir, "script")

	logger := log.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	commitID := report.Crashes[0].KernelSourceCommit
	getCmd := exec.Command(filepath.Join(scriptDir, "get.sh"), commitID)
	getCmd.Stdout = io.MultiWriter(stdout, logger.Writer())
	getCmd.Stderr = io.MultiWriter(stderr, logger.Writer())
	if err := getCmd.Run(); err != nil {
		return err
	}

	return nil
}
