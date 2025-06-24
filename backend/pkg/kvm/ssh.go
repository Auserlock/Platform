package kvm

import (
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// SSHManager ssh connection manager
type SSHManager struct {
	client  *ssh.Client
	session *ssh.Session
	config  SSHConfig
}

// SSHConfig config of ssh connection
type SSHConfig struct {
	Host    string
	Port    int
	User    string
	Passwd  string
	KeyPath string
	Timeout time.Duration
}

func NewSSHManager(config SSHConfig) *SSHManager {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SSHManager{
		config: config,
	}
}

func (sm *SSHManager) Connect() error {
	clientConfig := &ssh.ClientConfig{
		User:            sm.config.User,
		Timeout:         sm.config.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if sm.config.Passwd != "" {
		clientConfig.Auth = append(clientConfig.Auth, ssh.Password(sm.config.Passwd))
	}

	if sm.config.KeyPath != "" {
		if key, err := sm.loadPrivateKey(sm.config.KeyPath); err != nil {
			clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(key))
		} else {
			log.Errorf("fail to load private key: %v\n", err)
		}
	}

	address := fmt.Sprintf("%s:%d", sm.config.Host, sm.config.Port)
	maxRetries := 5
	var client *ssh.Client
	for i := 1; i <= maxRetries; i++ {
		var err error
		client, err = ssh.Dial("tcp", address, clientConfig)
		if err != nil {
			log.Errorf("fail to dial at try %d: %v", i, err)
		}
		time.Sleep(5 * time.Second)
	}

	sm.client = client
	log.Infof("SSH connection successful: %s\n", address)
	return nil
}

func (sm *SSHManager) loadPrivateKey(keyPath string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("fail to load private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("fail to parse private key: %v", err)
	}

	return signer, nil
}

func (sm *SSHManager) Execute(cmd string) (string, error) {
	if sm.client == nil {
		return "", fmt.Errorf("SSH client has not been initialized")
	}

	session, err := sm.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("fail to create session: %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("fail to execute command: %v", err)
	}
	log.Infof("command output: %s", string(output))
	return string(output), nil
}

func (sm *SSHManager) ExecutePersistent(cmd string) error {
	if sm.client == nil {
		return fmt.Errorf("SSH client has not been initialized")
	}

	session, err := sm.client.NewSession()
	if err != nil {
		return fmt.Errorf("fail to create session: %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()

	if err = session.Start(cmd); err != nil {
		return fmt.Errorf("fail to execute command: %v", err)
	}

	go func() {
		_, err := io.Copy(os.Stdout, stdout)
		if err != nil {
			log.Errorln(err)
		}
	}()
	go func() {
		_, err := io.Copy(os.Stderr, stderr)
		if err != nil {
			log.Errorln(err)
		}
	}()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("fail to execute command: %v", err)
	}

	return nil
}

func (sm *SSHManager) Close() error {
	if sm.session != nil {
		err := sm.session.Close()
		if err != nil {
			return err
		}
	}
	if sm.client != nil {
		return sm.client.Close()
	}
	return nil
}
