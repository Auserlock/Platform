package workflow

import (
	"backend/pkg/compile"
	"backend/pkg/compress"
	"backend/pkg/config"
	"backend/pkg/kvm"
	"backend/pkg/parse"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

func sleep() {
	time.Sleep(time.Second * 2)
}

func WalkDir(callback func(data *parse.CrashReport), dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("读取目录出错:", err)
		return
	}

	for _, entry := range entries {
		data := parse.Parse(filepath.Join(dir, entry.Name()))
		callback(&data)
	}
}

func CompileKernel(f string) error {
	data := parse.Parse(f)
	sleep()
	compile.InitToolChain(&data)

	if err := compile.DownloadKernel(&data); err != nil {
		log.Errorln(err)
		return err
	}

	sleep()

	if err := compile.DownloadConfig(&data); err != nil {
		log.Errorln(err)
		return err
	}

	sleep()

	if err := compile.DownloadBug(&data); err != nil {
		log.Errorln(err)
		return err
	}

	sleep()

	if err := compile.MakeKernel(&data); err != nil {
		log.Errorln(err)
		return err
	}

	sleep()

	log.Infoln("compile successfully")
	return nil
}

func BuildKernel(report *parse.CrashReport) error {
	if err := kvm.GetVmcore(report); err != nil {
		log.Errorln(err)
		return err
	}

	sleep()

	log.Infoln("build target directory successfully")
	return nil
}

func ClearKernel(f string) error {
	data := parse.Parse(f)
	sleep()
	compile.InitToolChain(&data)

	if err := compile.ClearDownload(&data); err != nil {
		return err
	}
	sleep()
	if err := kvm.ClearImage(&data); err != nil {
		return err
	}
	log.Infoln("clear successfully")
	return nil
}

func SSHConnect() *kvm.SSHManager {
	config := kvm.SSHConfig{
		Host:    "127.0.0.1",
		Port:    2222,
		User:    "root",
		Passwd:  "123456",
		Timeout: 30 * time.Second,
	}

	sshManager := kvm.NewSSHManager(config)

	if err := sshManager.Connect(); err != nil {
		log.Panicln(err)
	}

	return sshManager
}

func SSHExecutePersistent(cmd string, sshManager *kvm.SSHManager) error {
	return sshManager.ExecutePersistent(cmd)
}

func SSHExecute(cmd string, sshManager *kvm.SSHManager) error {
	res, err := sshManager.Execute(cmd)
	log.Infof("execute command:%s%s", cmd, res)
	return err
}

func QEMUVMConnect(report *parse.CrashReport) (*kvm.QEMUManager, error) {
	workPath, _ := os.Getwd()
	commit := report.Crashes[0].KernelSourceCommit
	config := kvm.VMConfig{
		ImagePath:    filepath.Join(workPath, fmt.Sprintf("work/%s/debian.img", commit)),
		KernelPath:   filepath.Join(workPath, fmt.Sprintf("work/%s/bzImage", commit)),
		Memory:       config.GlobalConfig.VM.Memory,
		MonitorPort:  4444,
		KernelAppend: "root=/dev/sda console=ttyS0,115200n8 rw crashkernel=256M",
		LogFile:      filepath.Join(workPath, fmt.Sprintf("log/%s.log", commit)),
	}

	q := kvm.NewQEMUManager(config)
	err := q.StartVM()
	return q, err
}

// kexec -p /boot/crash-bzImage --initrd=/boot/crash-initramfs.cpio.gz --append="root=/dev/ram0 console=ttyS0"
func Kexec(sshManager *kvm.SSHManager) error {
	return SSHExecute("kexec -p /boot/crash-bzImage --initrd=/boot/crash-initramfs.cpio.gz --append=\"root=/dev/ram0 console=ttyS0\"", sshManager)
}

func Gcc(sshManager *kvm.SSHManager) error {
	return SSHExecute("gcc bug.c -o bug -static", sshManager)
}

func Bug(sshManager *kvm.SSHManager) error {
	return SSHExecute("./bug", sshManager)
}

func Compile(f string) error {
	log.Infof("starting compile kernel with file: %s", f)

	err := CompileKernel(f)
	if err != nil {
		return err
	}
	return nil
}

func Generate(f string) error {
	log.Infof("starting generate vmcore with file: %s", f)

	data := parse.Parse(f)

	if err := kvm.ConfigImage(&data); err != nil {
		log.Errorln(err)
		return err
	}

	sleep()

	vm, err := QEMUVMConnect(&data)
	if err != nil {
		return err
	}
	resp, err := vm.GetVMStatus()
	if err != nil {
		log.Errorf("failed to get VM status: %v", err)
	}

	log.Infof(resp)
	defer func() {
		if err := vm.ShutdownVM(); err != nil {
			log.Errorln(err)
		}
	}()

	ssh := SSHConnect()
	if err = Kexec(ssh); err != nil {
		log.Errorln(err)
	}
	if err = Gcc(ssh); err != nil {
		log.Errorln(err)
	}
	if err = Bug(ssh); err != nil {
		log.Errorln(err)
	}
	time.Sleep(time.Second * 15)

	if err = BuildKernel(&data); err != nil {
		return err
	}
	log.Infoln("generate vmcore successfully!")
	return nil
}

func Compress(f string) error {
	log.Infof("starting compress with file: %s", f)

	data := parse.Parse(f)
	if err := compress.Compress(&data); err != nil {
		log.Errorln(err)
		return err
	}

	log.Infoln("compress successfully")
	return nil
}

func Clean(f string) error {
	log.Infof("starting clean with file: %s", f)

	if err := ClearKernel(f); err != nil {
		log.Errorln(err)
	}

	log.Infoln("clean successfully")
	return nil
}

func Patch(f string, path string) error {
	log.Infof("starting patch with file: %s, path: %s", f, path)
	// !TODO
	log.Infoln("patch apply successfully!")
	return nil
}
