package compress

// compress vmcore and linux build dir for analyse

import (
	"backend/pkg/parse"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func kernelPath(report *parse.CrashReport) string {
	rootPath, _ := os.Getwd()
	return filepath.Join(rootPath, fmt.Sprintf("build/%s/linux-%s", report.Crashes[0].KernelSourceCommit, report.Crashes[0].KernelSourceCommit))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func Compress(report *parse.CrashReport) error {
	kernelPath := kernelPath(report)
	kernelDir := filepath.Dir(kernelPath)
	kernelBase := filepath.Base(kernelPath)
	outputName := kernelBase + ".tar.zst"
	outputPath := filepath.Join(kernelDir, outputName)

	if fileExists(outputName) {
		return nil
	}

	log.Infof("compressing into tar.zst file: %s", outputPath)

	cmd := exec.Command("sudo", "tar",
		"-I", "zstd -19 -T0",
		"-cvf", outputName,
		kernelBase,
	)
	cmd.Dir = kernelDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar compress failed: %w", err)
	}

	stat, err := os.Stat(outputPath)
	if err != nil {
		return err
	}
	log.Infof("compress successfully, size of tar.zst file：%.2f GB", float64(stat.Size())/1024/1024/1024)

	return nil
}
