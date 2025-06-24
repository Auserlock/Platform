package main

import (
	"backend/pkg/parse"
	"backend/pkg/workflow"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

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
}

func work(f string) int {
	log.Printf("Starting work with file: %s", f)

	data := parse.Parse(f)
	err := workflow.CompileKernel(f)
	if err != nil {
		log.Errorln(err)
		return 1
	}
	vm, err := workflow.QEMUVMConnect(&data)
	if err != nil {
		log.Errorln(err)
		return 1
	}
	resp, err := vm.GetVMStatus()
	if err != nil {
		log.Errorln(err)
	}
	fmt.Println(resp)
	defer func() {
		if err := vm.ShutdownVM(); err != nil {
			log.Errorln(err)
		}
	}()

	ssh := workflow.SSHConnect()
	if err = workflow.Kexec(ssh); err != nil {
		log.Errorln(err)
	}
	if err = workflow.Gcc(ssh); err != nil {
		log.Errorln(err)
	}
	if err = workflow.Bug(ssh); err != nil {
		log.Errorln(err)
	}
	time.Sleep(time.Second * 10)
	if err = workflow.BuildKernel(&data); err != nil {
		log.Errorln(err)
	}
	log.Printf("Work finished for file: %s", f) // 增加完成日志
	return 0
}

func clean(f string) {
	log.Printf("Starting work with file: %s", f)

	if err := workflow.ClearKernel(f); err != nil {
		log.Errorln(err)
	}
	log.Printf("Work finished for file: %s", f)
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run main.go <json_file_path>")
	}

	taskType := os.Args[1]

	switch taskType {
	case "kernel-build":
		if work(os.Args[2]) != 0 {
			os.Exit(1)
		}
		break
	case "patch-apply":
		break
	case "clean":
		clean(os.Args[2])
		break
	}
	return
}
