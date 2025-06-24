package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

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

func (cr *CrashReport) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed, got %T instead", value)
	}
	if bytes == nil {
		return nil
	}
	return json.Unmarshal(bytes, cr)
}

func (cr *CrashReport) Value() (driver.Value, error) {
	return json.Marshal(cr)
}
