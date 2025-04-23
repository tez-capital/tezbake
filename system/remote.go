package system

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	"github.com/AlecAivazis/survey/v2"
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
	pw := ""
	prompt := &survey.Password{
		Message: reason,
	}
	err := survey.AskOne(prompt, &pw)
	// bytepw, err := term.ReadPassword(int(syscall.Stdin))
	util.AssertE(err, failureMsg)
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
			escapedValue := fmt.Sprintf("%q", v)
			result += fmt.Sprintf("export %s=%s;", k, escapedValue)
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

func RunSshCommandWithOutputChannel(client *ssh.Client, cmd string, env *map[string]string, outputChannel chan<- string) *SshCommandResult {
	session, err := client.NewSession()
	if err != nil {
		return &SshCommandResult{
			Error:    err,
			ExitCode: -1,
		}
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

	var wg sync.WaitGroup
	// Increment the WaitGroup counter for each goroutine
	wg.Add(2)
	go func() {
		defer wg.Done()
		// feed the output channel with the output of the command
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Send each line of output to the channel
			outputChannel <- scanner.Text()
		}
	}()
	go func() {
		defer wg.Done()
		// feed the output channel with the output of the command
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			// Send each line of output to the channel
			outputChannel <- scanner.Text()
		}
	}()

	cmd = buildUpEnv(env) + cmd

	exitCode := 0

	err = session.Run(cmd)
	if err != nil {
		exitCode = -1
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}
	}
	wg.Wait()
	return &SshCommandResult{
		Error:    err,
		ExitCode: exitCode,
	}
}

// func SSHCopyFile(srcPath, dstPath string) error {
// 	config := &ssh.ClientConfig{
// 		User: "user",
// 		Auth: []ssh.AuthMethod{
// 			ssh.Password("pass"),
// 		},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}
// 	fmt.Println()
// 	client, _ := ssh.Dial("tcp", "remotehost:22", config)
// 	defer client.Close()

// 	// open an SFTP session over an existing ssh connection.
// 	sftp, err := sftp.NewClient(client)
// 	if err != nil {
// 		return err
// 	}
// 	defer sftp.Close()

// 	// Open the source file
// 	srcFile, err := os.Open(srcPath)
// 	if err != nil {
// 		return err
// 	}
// 	defer srcFile.Close()

// 	// Create the destination file
// 	dstFile, err := sftp.Create(dstPath)
// 	if err != nil {
// 		return err
// 	}
// 	defer dstFile.Close()

// 	// write to file
// 	if _, err := dstFile.ReadFrom(srcFile); err != nil {
// 		return err
// 	}
// 	return nil
// }

/*
generate ssh key
inject ssh key to server under --user
(add user if needed)
setenv
get remote platform
download target platform locally
transfer over ssh

setup:
- check whether key or password
- test connectivity
- --remote-auth=pass/key
- --remote-node=user@addr

- --remote-elevate=su 	(REMOTE_SU_USER, REMOTE_SU_PASS)
- --remote-elevate=sudo (REMOTE_SUDO_PASS)
- --remote-elevate=none (DEFAULT, connecting as root)


*/
