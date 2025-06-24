package parse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const SyzkallerURL = "https://syzkaller.appspot.com"

type FixCommit struct {
	Title  string `json:"title"`
	Link   string `json:"link"`
	Hash   string `json:"hash"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}
type Crash struct {
	Title               string `json:"title"`
	SyzReproducer       string `json:"syz-reproducer"`
	CReproducer         string `json:"c-reproducer"`
	KernelConfig        string `json:"kernel-config"`
	KernelSourceGit     string `json:"kernel-source-git"`
	KernelSourceCommit  string `json:"kernel-source-commit"`
	SyzkallerGit        string `json:"syzkaller-git"`
	SyzkallerCommit     string `json:"syzkaller-commit"`
	CompilerDescription string `json:"compiler-description"`
	Architecture        string `json:"architecture"`
	CrashReportLink     string `json:"crash-report-link"`
}
type CrashReport struct {
	Version           int         `json:"version"`
	Title             string      `json:"title"`
	DisplayTitle      string      `json:"display-title"`
	ID                string      `json:"id"`
	Status            string      `json:"status"`
	FixCommits        []FixCommit `json:"fix-commits"`
	Discussions       []string    `json:"discussions"`
	Crashes           []Crash     `json:"crashes"`
	Subsystems        []string    `json:"subsystems"`
	ParentOfFixCommit string      `json:"parent_of_fix_commit"`
	Patch             string      `json:"patch"`
	PatchModified     []string    `json:"patch_modified_files"`
}

// kernelPath Construct directory with CrashReport
func kernelPath(report *CrashReport) string {
	return fmt.Sprintf("build/%s/linux-%s", report.Crashes[0].KernelSourceCommit, report.Crashes[0].KernelSourceCommit)
}

// Parse panic program when parse json error
func Parse(filename string) CrashReport {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Panicln(err)
	}
	var report CrashReport
	err = json.Unmarshal(data, &report)
	if err != nil {
		log.Panicln(err)
	}
	log.Infof("parse crash report %s success", filename)
	return report
}

// WritePatch write patch into linux kernel dir, use git apply to apply patch and rebuild kernel
func WritePatch(report *CrashReport, filename string) {
	filePath := filepath.Join(kernelPath(report), filename)
	err := os.WriteFile(filePath, []byte(report.Patch), 0644)
	if err != nil {
		log.Errorln(err)
		return
	}
	log.Infof("patch written to %s\n", filename)
}
