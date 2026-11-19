package system

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"strings"

	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	"github.com/kballard/go-shellquote"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	SSH_MODE_PASS = "pass"
	SSH_MODE_KEY  = "key"
)

type SshConnectionDetails struct {
	Username string
	Host     string
	Port     string
}

type SshCommandResult struct {
	Stdout   []byte
	Stderr   []byte
	Error    error
	ExitCode int
}

func promptForPassword(reason string, failureMsg string) []byte {
	pw := util.RequirePasswordE(reason, failureMsg, constants.ExitInternalError)
	// bytepw, err := term.ReadPassword(int(syscall.Stdin))
	return []byte(pw)
}

func GetRemoteConnectionDetails(remote string) *SshConnectionDetails {
	sshUser := constants.DefaultSshUser
	sshAddr := remote
	sshPort := "22"
	if parts := strings.Split(remote, "@"); len(parts) > 1 {
		sshUser = parts[0]
		sshAddr = parts[1]
	}
	if parts := strings.Split(sshAddr, ":"); len(parts) > 1 {
		sshAddr = parts[0]
		sshPort = parts[1]
	}
	return &SshConnectionDetails{
		Username: sshUser,
		Host:     sshAddr,
		Port:     sshPort,
	}
}

func OpenSshSessionS(connectionDetails *SshConnectionDetails, mode string, privateKeyOrPassword []byte) (*ssh.Client, *sftp.Client, error) {
	config := &ssh.ClientConfig{
		User:            connectionDetails.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	switch mode {
	case SSH_MODE_KEY:
		var key ssh.Signer
		var err error
		key, err = ssh.ParsePrivateKey([]byte(privateKeyOrPassword))
		if err != nil && err.Error() == "ssh: this private key is passphrase protected" {
			pass := []byte(os.Getenv("REMOTE_KEY_PASS"))
			if len(pass) == 0 {
				if len(pass) == 0 {
					pass = promptForPassword("Please provide password for ssh key:", "Failed to get decrypt ssh key!")
				}
			}
			key, err = ssh.ParsePrivateKeyWithPassphrase(privateKeyOrPassword, pass)
		}
		if err != nil {
			return nil, nil, err
		}
		config.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(key),
		}
	case SSH_MODE_PASS:
		if len(privateKeyOrPassword) == 0 {
			privateKeyOrPassword = []byte(os.Getenv("REMOTE_PASS"))
			if len(privateKeyOrPassword) == 0 {
				privateKeyOrPassword = promptForPassword("Please provide password for remote login:", "Failed to get password for ssh login!")
			}
		}
		config.Auth = []ssh.AuthMethod{
			ssh.Password(string(privateKeyOrPassword)),
		}
	default:
		return nil, nil, errors.New("unsupported ssh auth method")
	}
	client, err := ssh.Dial("tcp", net.JoinHostPort(connectionDetails.Host, connectionDetails.Port), config)
	if err != nil {
		return nil, nil, err
	}
	sftp, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, sftp, nil
}

func OpenSshSession(connectionDetails *SshConnectionDetails, mode string, privateKeyOrPassword []byte) (*ssh.Client, *sftp.Client) {
	sshClient, sftp, err := OpenSshSessionS(connectionDetails, mode, privateKeyOrPassword)
	util.AssertE(err, "Failed to create ssh session!")

	return sshClient, sftp
}

func buildUpEnv(env *map[string]string) string {
	result := ""
	if env != nil {
		for k, v := range *env {
			result += "export " + shellquote.Join(k+"="+v) + ";"
		}
	}
	return result
}

func RunSshCommand(client *ssh.Client, cmd string, env *map[string]string) *SshCommandResult {
	var stdout, stderr bytes.Buffer
	session, err := client.NewSession()
	if err != nil {
		return &SshCommandResult{
			Error:    err,
			ExitCode: -1,
		}
	}
	defer session.Close()
	session.Stdout = &stdout
	session.Stderr = &stderr

	cmd = buildUpEnv(env) + cmd

	exitCode := 0
	err = session.Run(cmd)
	if err != nil {
		exitCode = -1
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}
	}

	return &SshCommandResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		Error:    err,
		ExitCode: exitCode,
	}
}

func RunPipedSshCommand(client *ssh.Client, cmd string, env *map[string]string) *SshCommandResult {
	session, err := client.NewSession()
	if err != nil {
		return &SshCommandResult{
			Error:    err,
			ExitCode: -1,
		}
	}
	defer session.Close()
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	cmd = buildUpEnv(env) + cmd

	exitCode := 0

	err = session.Run(cmd)
	if err != nil {
		exitCode = -1
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}
	}
	return &SshCommandResult{
		Error:    err,
		ExitCode: exitCode,
	}
}

// ForwardOutputLines streams complete lines without Scanner's 64 KiB token limit.
func ForwardOutputLines(src io.Reader, outputChannel chan<- string) error {
	reader := bufio.NewReader(src)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			outputChannel <- strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func RunSshCommandWithOutputChannel(client *ssh.Client, cmd string, env *map[string]string, outputChannel chan<- string) *SshCommandResult {
	session, err := client.NewSession()
	if err != nil {
		return &SshCommandResult{Error: err, ExitCode: -1}
	}
	defer session.Close()
	stdout, err := session.StdoutPipe()
	if err != nil {
		return &SshCommandResult{Error: err, ExitCode: -1}
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return &SshCommandResult{Error: err, ExitCode: -1}
	}
	if err := session.Start(buildUpEnv(env) + cmd); err != nil {
		return &SshCommandResult{Error: err, ExitCode: -1}
	}
	outputErrors := make(chan error, 2)
	go func() { outputErrors <- ForwardOutputLines(stdout, outputChannel) }()
	go func() { outputErrors <- ForwardOutputLines(stderr, outputChannel) }()
	outputErr := errors.Join(<-outputErrors, <-outputErrors)
	err = errors.Join(session.Wait(), outputErr)
	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitErr *ssh.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitStatus()
		}
	}
	return &SshCommandResult{Error: err, ExitCode: exitCode}
}
