package main

import (
	"backend/pkg/compile"
	"backend/pkg/config"
	"backend/pkg/parse"
	"backend/pkg/workflow"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

var cnt map[string]int = make(map[string]int)
var same []string

var (
	taskType   string
	jsonPath   string
	patchPath  string
	doCompile  bool
	doGenerate bool
	doCompress bool
	configs    ConfigMap
)

func stat(report *parse.CrashReport) {
	commit := report.Crashes[0].KernelSourceCommit
	cnt[commit]++
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       false,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
		QuoteEmptyFields:       false,
		DisableQuote:           true,
		ForceColors:            true,
	})

	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)

	log.SetLevel(log.WarnLevel)
	dir := "/home/arch/TraceGPT/build-vmcore/kernel-benchmark"
	workflow.WalkDir(stat, dir)
	for k, v := range cnt {
		if v > 1 {
			same = append(same, k)
		}
	}
	log.SetLevel(log.DebugLevel)

	flag.StringVar(&taskType, "type", "", "task type: kernel-build / patch-apply / clean")
	flag.StringVar(&taskType, "t", "", "shorthand for --type")

	flag.StringVar(&jsonPath, "file", "", "task file path")
	flag.StringVar(&jsonPath, "f", "", "shorthand for --file")

	flag.StringVar(&patchPath, "patch", "", "patch file path")
	flag.StringVar(&patchPath, "p", "", "shorthand for --patch")

	flag.BoolVar(&doCompile, "compile", false, "compile kernel")
	flag.BoolVar(&doCompile, "c", false, "shorthand for --compile")

	flag.BoolVar(&doGenerate, "generate", false, "generate vmcore")
	flag.BoolVar(&doGenerate, "g", false, "shorthand for --generate")

	flag.BoolVar(&doCompress, "compress", false, "compress vmcore and linux kernel dir")
	flag.BoolVar(&doCompress, "z", false, "shorthand for --compress")

	flag.Var(&configs, "config", "override kernel config, e.g. --config CONFIG_KASAN=y")

	err := config.Load("config.json")
	if err != nil {
		log.Panicf("Failed to load config: %v", err)
	}
}

type ConfigMap map[string]string

func (cm *ConfigMap) String() string {
	return fmt.Sprintf("%v", *cm)
}

func (cm *ConfigMap) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid config: %s (expect CONFIG_XXX=y)", value)
	}
	key := parts[0]
	val := parts[1]
	if *cm == nil {
		*cm = make(map[string]string)
	}
	(*cm)[key] = val
	return nil
}

func contain(s string) bool {
	for _, v := range same {
		if v == s {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()

	for k, v := range configs {
		compile.ModifyConfig(k, v)
	}

	switch taskType {
	case "kernel-build":
		if doCompile {
			data := parse.Parse(jsonPath)
			if contain(data.Crashes[0].KernelSourceCommit) {
				rootPath, _ := os.Getwd()
				targetDir := filepath.Join(rootPath, fmt.Sprintf("build/%s", data.Crashes[0].KernelSourceCommit))

				fmt.Printf("commit %s  is duplicated and may overwrite the current build folder. Continuing will delete the previously built task.\n", data.Crashes[0].KernelSourceCommit)
				fmt.Print("enter y to continue, others to cancel: ")

				var input string
				fmt.Scanln(&input)

				if strings.ToLower(input) != "y" {
					fmt.Println("operation cancelled.")
					os.Exit(0)
				}

				err := os.RemoveAll(targetDir)
				if err != nil {
					log.Panicln(err)
				}

				fmt.Println("previous build folder deleted successfully.")
			}
			err := workflow.Compile(jsonPath)
			if err != nil {
				log.Errorf("Failed to compile kernel: %v", err)
				os.Exit(1)
			}
		}
		if doGenerate {
			err := workflow.Generate(jsonPath)
			if err != nil {
				log.Errorf("Failed to generate vmcore: %v", err)
				os.Exit(1)
			}
		}
		if doCompress {
			err := workflow.Compress(jsonPath)
			if err != nil {
				log.Errorf("Failed to compress: %v", err)
				os.Exit(1)
			}
		}
		if !doCompile && !doGenerate && !doCompress {
			log.Error("No action specified. Use --compile, --generate, or --compress.")
			os.Exit(1)
		}
	case "clean":
		err := workflow.Clean(jsonPath)
		if err != nil {
			log.Errorf("Failed to clean: %v", err)
			os.Exit(1)
		}
	case "patch-apply":
		err := workflow.Patch(jsonPath, patchPath)
		if err != nil {
			log.Errorf("Failed to apply patch: %v", err)
			os.Exit(1)
		}
	}
}
