package kvm

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type QEMUManager struct {
	cmd         *exec.Cmd
	monitorConn net.Conn
	SSH         *SSHManager
	vmConfig    VMConfig
}

type VMConfig struct {
	ImagePath    string
	KernelPath   string
	Memory       string
	MonitorPort  int
	KernelAppend string
	LogFile      string
}

func NewQEMUManager(config VMConfig) *QEMUManager {
	return &QEMUManager{
		vmConfig: config,
	}
}

func (qm *QEMUManager) StartVM() error {
	args := []string{
		"-m", qm.vmConfig.Memory,
		"-kernel", qm.vmConfig.KernelPath,
		"-drive", fmt.Sprintf("file=%s,format=raw,if=ide", qm.vmConfig.ImagePath),
		"-append", qm.vmConfig.KernelAppend,
		"-device", "virtio-net,netdev=net0",
		"-netdev", "user,id=net0,hostfwd=tcp::2222-:22",
		"-serial", fmt.Sprintf("file:%s", qm.vmConfig.LogFile),
	}
	args = append(args, "-nographic")
	args = append(args, "-enable-kvm")
	args = append(args, "-cpu", "host,-x2apic")
	args = append(args, "-no-reboot")
	args = append(args, "-monitor", fmt.Sprintf("tcp:127.0.0.1:%d,server,wait", qm.vmConfig.MonitorPort))

	qm.cmd = exec.Command("qemu-system-x86_64", args...)

	log.Infof("start qemu vm, command as follow:\n%s\n", strings.Join(args, " "))
	if err := qm.cmd.Start(); err != nil {
		return err
	}
	time.Sleep(time.Second * 30)

	if err := qm.connectMonitor(); err != nil {
		return fmt.Errorf("fail to connect monitor port: %v", err)
	}

	log.Infoln("start qemu vm successfully, pid =", qm.cmd.Process.Pid)
	return nil
}

func (qm *QEMUManager) connectMonitor() error {
	var err error
	for i := 0; i < 10; i++ {
		qm.monitorConn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", qm.vmConfig.MonitorPort))
		if err == nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("cannot connect monitor port: %v", err)
	}

	reader := bufio.NewReader(qm.monitorConn)
	_, err = reader.ReadString('\n')
	return err
}

var monitorMutex sync.Mutex

func drainMonitor(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	io.Copy(io.Discard, conn)
	conn.SetReadDeadline(time.Time{}) // 清除deadline
}

// ExecuteMonitorCommand !TODO debug this one to display better output
func (qm *QEMUManager) ExecuteMonitorCommand(command string) (string, error) {
	if qm.monitorConn == nil {
		return "", fmt.Errorf("monitor connection not established")
	}

	monitorMutex.Lock()
	defer monitorMutex.Unlock()

	// 先清空旧的残留输出，避免回显叠加
	drainMonitor(qm.monitorConn)

	_, err := fmt.Fprintf(qm.monitorConn, "%s\n", command)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(qm.monitorConn)
	var response strings.Builder

	for {
		qm.monitorConn.SetReadDeadline(time.Now().Add(3 * time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			return "", fmt.Errorf("failed reading response: %w", err)
		}

		fmt.Print(line) // 打印调试用

		response.WriteString(line)

		if strings.HasPrefix(line, "(qemu)") {
			break
		}
	}

	return response.String(), nil
}

func (qm *QEMUManager) ConnectVM(config SSHConfig) error {
	var conn net.Conn
	var err error

	for i := 0; i < 30; i++ {
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", qm.vmConfig.MonitorPort), 5*time.Second)
		if err == nil {
			err := conn.Close()
			if err != nil {
				return err
			}
			break
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("ssh systemctl not available: %v", err)
	}

	sshManager := NewSSHManager(config)

	err = sshManager.Connect()

	qm.SSH = sshManager

	return err
}

func (qm *QEMUManager) GetVMStatus() (string, error) {
	return qm.ExecuteMonitorCommand("info status")
}

func (qm *QEMUManager) PauseVM() (string, error) {
	return qm.ExecuteMonitorCommand("stop")
}

func (qm *QEMUManager) ResumeVM() (string, error) {
	return qm.ExecuteMonitorCommand("cont")
}

func (qm *QEMUManager) ShutdownVM() error {
	log.Infoln("shutdown qemu vm...")

	if qm.monitorConn != nil {
		_, err := qm.ExecuteMonitorCommand("system_powerdown")
		if err != nil {
			log.Errorln(err)
		}
		time.Sleep(10 * time.Second)
	}

	if qm.SSH != nil {
		err := qm.SSH.Close()
		if err != nil {
			return err
		}
	}

	if qm.monitorConn != nil {
		err := qm.monitorConn.Close()
		if err != nil {
			return err
		}
	}

	if qm.cmd != nil && qm.cmd.Process != nil {
		_, err := qm.cmd.Process.Wait()
		if err != nil {
			return err
		}
	}

	log.Infoln("vm shutdown successfully")
	return nil
}
