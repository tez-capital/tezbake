package ami

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kballard/go-shellquote"
	"github.com/pkg/sftp"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"golang.org/x/crypto/ssh"
)

func writeTestFile(t *testing.T, fileName string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(fileName, data, mode); err != nil {
		t.Fatal(err)
	}
}

func writeTestLocator(t *testing.T, appDir string, locator *RemoteConfiguration) {
	t.Helper()
	data, err := json.Marshal(locator)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(appDir, LocatorFile), data, 0644)
}

// The test server uses SFTP against temporary files and emulates the required SSH commands.
func newRemoteTestServer(t *testing.T) (string, *RemoteConfiguration, *atomic.Int32) {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	serverConfig := &ssh.ServerConfig{NoClientAuth: true}
	serverConfig.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	localApp := t.TempDir()
	remoteInstance := filepath.Join(t.TempDir(), "remote instance ' $(echo ignored)")
	if err := os.MkdirAll(remoteInstance, 0755); err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	locator := &RemoteConfiguration{
		App: "node", Host: host, Port: port, Username: "test-user", InstancePath: remoteInstance,
		PrivateKey: filepath.Join(localApp, "private.key"), PublicKey: filepath.Join(localApp, "public.key"),
	}
	privateKeyData, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, locator.PrivateKey, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyData}), 0600)
	writeTestFile(t, locator.PublicKey, ssh.MarshalAuthorizedKey(signer.PublicKey()), 0644)
	writeTestLocator(t, localApp, locator)
	t.Cleanup(func() { ForgetRemoteConfiguration(localApp) })
	forwardExit := &atomic.Int32{}
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				server, channels, requests, err := ssh.NewServerConn(connection, serverConfig)
				if err != nil {
					connection.Close()
					return
				}
				defer server.Close()
				go ssh.DiscardRequests(requests)
				for incoming := range channels {
					channel, channelRequests, err := incoming.Accept()
					if err != nil {
						return
					}
					go func() {
						defer channel.Close()
						for request := range channelRequests {
							switch request.Type {
							case "subsystem":
								request.Reply(true, nil)
								sftpServer, err := sftp.NewServer(channel)
								if err != nil {
									return
								}
								sftpServer.Serve()
								sftpServer.Close()
								return
							case "exec":
								var payload struct{ Command string }
								ssh.Unmarshal(request.Payload, &payload)
								request.Reply(true, nil)
								exitCode, output, err := runRemoteTestCommand(payload.Command, locator, int(forwardExit.Load()))
								if err != nil {
									t.Errorf("remote test command: %v", err)
									exitCode = 1
									fmt.Fprintln(channel.Stderr(), err)
								}
								io.WriteString(channel, output)
								channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(exitCode)}))
								return
							default:
								request.Reply(false, nil)
							}
						}
					}()
				}
			}()
		}
	}()
	return localApp, locator, forwardExit
}

func runRemoteTestCommand(command string, locator *RemoteConfiguration, forwardExit int) (int, string, error) {
	args, err := shellquote.Split(command)
	if err != nil {
		return 1, "", err
	}
	if len(args) == 3 && args[0] == "mktemp" && args[1] == "--" && filepath.Base(args[2]) == ".tezbake-upload.XXXXXXXXXX" {
		file, err := os.CreateTemp(filepath.Dir(args[2]), ".tezbake-upload.*")
		if err != nil {
			return 1, "", err
		}
		if err := file.Close(); err != nil {
			os.Remove(file.Name())
			return 1, "", err
		}
		return 0, file.Name() + "\n", nil
	}
	if len(args) >= 2 && args[0] == "tezbake" && args[1] == "execute-ami" {
		index := slices.Index(args, "--app")
		if index < 0 || index+1 >= len(args) || args[index+1] != locator.GetRemoteAppPath() {
			return 1, "", fmt.Errorf("wrong remote app path in %q", command)
		}
		if forwardExit == 2 {
			return 2, "false\n", nil
		}
		return forwardExit, "true\n", nil
	}
	if len(args) != 5 || args[0] != "tezbake" || args[1] != "execute" || args[2] != "--elevate" || args[3] != "--base64" {
		return 1, "", fmt.Errorf("unexpected command %q", command)
	}
	decodedCommand, err := base64.StdEncoding.DecodeString(args[4])
	if err != nil {
		return 1, "", err
	}
	args, err = shellquote.Split(string(decodedCommand))
	if err != nil || len(args) != 4 {
		return 1, "", fmt.Errorf("unsafe folder/archive command %q", decodedCommand)
	}
	switch args[0] {
	case "mkdir":
		if args[1] != "-p" || args[2] != "--" || args[3] != locator.GetRemoteAppPath() {
			return 1, "", fmt.Errorf("unexpected mkdir target %q", args)
		}
		err = os.MkdirAll(args[3], 0755)
	case "chown":
		if args[1] != "--" || args[2] != "test-user:test-user" || args[3] != locator.GetRemoteAppPath() {
			return 1, "", fmt.Errorf("chown affected path outside the app: %q", args)
		}
	case "chmod":
		if args[1] != "0644" || args[2] != "--" {
			return 1, "", fmt.Errorf("unexpected archive permissions %q", args)
		}
		err = os.Chmod(args[3], 0644)
	default:
		err = fmt.Errorf("unexpected command %q", args)
	}
	return 0, "", err
}

type remoteUploadReader func([]byte) (int, error)

func (read remoteUploadReader) Read(p []byte) (int, error) { return read(p) }

func TestRemoteUploadStagesPrivately(t *testing.T) {
	for _, test := range []struct {
		name    string
		readErr error
	}{
		{name: "successful", readErr: io.EOF},
		{name: "interrupted", readErr: errors.New("source read failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, locator, _ := newRemoteTestServer(t)
			session, err := locator.OpenAppRemoteSession()
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()

			dir := locator.InstancePath
			destination := filepath.Join(dir, "config.txt")
			original := []byte("previous configuration")
			writeTestFile(t, destination, original, 0644)
			if err := os.Chmod(destination, 0644); err != nil {
				t.Fatal(err)
			}
			var uploaded []byte
			reader := remoteUploadReader(func(p []byte) (int, error) {
				staging, err := filepath.Glob(filepath.Join(dir, ".tezbake-upload.*"))
				if err != nil || len(staging) != 1 {
					t.Fatalf("expected one upload staging file: paths=%q error=%v", staging, err)
				}
				info, err := os.Stat(staging[0])
				if err != nil || info.Mode().Perm() != 0600 || info.Size() != int64(len(uploaded)) {
					t.Fatalf("unexpected staging file during upload: info=%v error=%v", info, err)
				}
				if data, err := os.ReadFile(destination); err != nil || !bytes.Equal(data, original) {
					t.Fatalf("destination changed during upload: data=%q error=%v", data, err)
				}
				if uploaded != nil {
					return 0, test.readErr
				}
				uploaded = bytes.Repeat([]byte{'x'}, len(p))
				return copy(p, uploaded), nil
			})
			const finalMode = 0400
			err = session.writeReaderToRemote(destination, reader, finalMode)
			if len(uploaded) == 0 {
				t.Fatalf("upload returned before reading from its source: %v", err)
			}
			wantContent, wantMode := uploaded, os.FileMode(finalMode)
			if test.readErr == io.EOF {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				if !errors.Is(err, test.readErr) {
					t.Fatalf("upload did not return the source error: %v", err)
				}
				wantContent, wantMode = original, 0644
			}
			if data, err := os.ReadFile(destination); err != nil || !bytes.Equal(data, wantContent) {
				t.Fatalf("unexpected destination contents: size=%d error=%v", len(data), err)
			}
			if info, err := os.Stat(destination); err != nil || info.Mode().Perm() != wantMode {
				t.Fatalf("expected destination mode %04o: info=%v error=%v", wantMode, info, err)
			}
			if staging, err := filepath.Glob(filepath.Join(dir, ".tezbake-upload.*")); err != nil || len(staging) != 0 {
				t.Fatalf("upload staging survived cleanup: paths=%q error=%v", staging, err)
			}
		})
	}
}

func TestRemoteExecutionAndConfigurationUseLocatorPath(t *testing.T) {
	localApp, locator, _ := newRemoteTestServer(t)
	if exitCode, err := Execute(localApp, "info"); err != nil || exitCode != 0 {
		t.Fatalf("Execute: code=%d error=%v", exitCode, err)
	}
	if output, exitCode, err := ExecuteGetOutput(localApp, "info"); err != nil || exitCode != 0 || output != "true\n" {
		t.Fatalf("ExecuteGetOutput: output=%q code=%d error=%v", output, exitCode, err)
	}
	output := make(chan string, 2)
	if exitCode, err := ExecuteWithOutputChannel(localApp, output, "info"); err != nil || exitCode != 0 {
		t.Fatalf("ExecuteWithOutputChannel: code=%d error=%v", exitCode, err)
	}
	close(output)
	if line := <-output; line != "true" {
		t.Fatalf("unexpected remote channel output %q", line)
	}
	if err := WriteFile(localApp, []byte("remote file"), "nested/config.txt"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(locator.GetRemoteAppPath(), "nested", "config.txt"))
	if err != nil || string(data) != "remote file" {
		t.Fatalf("WriteFile used wrong remote path: data=%q error=%v", data, err)
	}
	definition := map[string]any{"user": "restored-user"}
	outside := filepath.Join(t.TempDir(), "preserve")
	writeTestFile(t, outside, []byte("outside file"), 0644)
	if err := os.Symlink(outside, path.Join(locator.GetRemoteAppPath(), "app.json.new")); err != nil {
		t.Fatal(err)
	}
	if err := WriteAppDefinition(localApp, definition, "app.json"); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "outside file" {
		t.Fatalf("remote configuration write followed a temporary symlink: data=%q error=%v", data, err)
	}
	loaded, err := ReadAppDefinition(localApp, "app.json")
	if err != nil || loaded["user"] != "restored-user" {
		t.Fatalf("ReadAppDefinition: definition=%v error=%v", loaded, err)
	}
	_, definitionPath, err := FindAppDefinition(localApp)
	if err != nil || definitionPath != path.Join(locator.GetRemoteAppPath(), "app.json") {
		t.Fatalf("FindAppDefinition: path=%q error=%v", definitionPath, err)
	}
}

func TestIsAppInstalledSRemoteDistinguishesFailure(t *testing.T) {
	localApp, _, forwardExit := newRemoteTestServer(t)
	if installed, err := IsAppInstalledS(localApp); !installed || err != nil {
		t.Fatalf("remote installed app: installed=%v error=%v", installed, err)
	}
	forwardExit.Store(2)
	if installed, err := IsAppInstalledS(localApp); installed || err != nil {
		t.Fatalf("remote absent app: installed=%v error=%v", installed, err)
	}
	forwardExit.Store(22)
	if installed, err := IsAppInstalledS(localApp); installed || err == nil {
		t.Fatalf("remote failed probe: installed=%v error=%v", installed, err)
	}
	if err := os.Remove(filepath.Join(localApp, "private.key")); err != nil {
		t.Fatal(err)
	}
	if installed, err := IsAppInstalledS(localApp); installed || err == nil {
		t.Fatalf("remote missing SSH key: installed=%v error=%v", installed, err)
	}
}

func TestRemoteTempArchivePermissionsAndCleanup(t *testing.T) {
	_, locator, _ := newRemoteTestServer(t)
	session, err := locator.OpenAppRemoteSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	archivePath, err := session.CreateRemoteTempArchive()
	if err != nil {
		t.Fatal(err)
	}
	defer session.RemoveRemoteTempArchive(archivePath)
	dir := filepath.Dir(archivePath)
	info, err := os.Stat(dir)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("temporary archive directory is not private: info=%v error=%v", info, err)
	}
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("archive path must remain absent until pack: %v", err)
	}
	writeTestFile(t, archivePath, []byte("packed app"), 0600)
	if err := session.MakeRemoteArchiveReadable(archivePath); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(archivePath)
	if err != nil || info.Mode().Perm() != 0644 {
		t.Fatalf("archive cannot be downloaded: info=%v error=%v", info, err)
	}
	destination := filepath.Join(t.TempDir(), "download.zip")
	if err := session.DownloadFile(archivePath, destination, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "packed app" {
		t.Fatalf("download: data=%q error=%v", data, err)
	}
	writeTestFile(t, filepath.Join(dir, "zip-writer-residue"), nil, 0600)
	if err := session.RemoveRemoteTempArchive(archivePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("temporary archive directory survived cleanup: %v", err)
	}
	if err := session.RemoveRemoteTempArchive(archivePath); err != nil {
		t.Fatalf("cleanup must be idempotent: %v", err)
	}
	if err := session.MakeRemoteArchiveReadable("/tmp/unrelated.zip"); err == nil {
		t.Fatal("accepted an archive outside the private staging directory")
	}
}

func TestSaveRemoteElevationCredentialsAndInvalidate(t *testing.T) {
	appDir := t.TempDir()
	t.Cleanup(func() { ForgetRemoteConfiguration(appDir) })
	passwordCache := slices.Clone(elevationCredentialsPasswordCache)
	t.Cleanup(func() { elevationCredentialsPasswordCache = passwordCache })
	config := &RemoteConfiguration{App: "node", Elevate: REMOTE_ELEVATION_SUDO}
	credentials := &RemoteElevateCredentials{Password: "new password"}
	if err := SaveRemoteElevationCredentials(appDir, config, credentials, "encryption password"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(appDir, ElevationCredentialsEncFile))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("unsafe credential permissions: info=%v error=%v", info, err)
	}
	ForgetRemoteConfiguration(appDir)
	config.ElevationCredentials = nil
	loaded, err := config.GetElevationCredentials()
	if err != nil || loaded.Password != "new password" || loaded.Kind != REMOTE_ELEVATION_SUDO {
		t.Fatalf("encrypted credentials: credentials=%v error=%v", loaded, err)
	}
	if slices.Contains(elevationCredentialsPasswordCache, "") {
		t.Fatal("cache reuse added an empty encryption password")
	}
	credentials.Password = "plain password"
	if err := SaveRemoteElevationCredentials(appDir, config, credentials, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(appDir, ElevationCredentialsEncFile)); !os.IsNotExist(err) {
		t.Fatalf("stale encrypted credentials survived replacement: %v", err)
	}
	ForgetRemoteConfiguration(appDir)
	config.ElevationCredentials = nil
	loaded, err = config.GetElevationCredentials()
	if err != nil || loaded.Password != "plain password" || loaded.Kind != REMOTE_ELEVATION_SUDO {
		t.Fatalf("plain credentials: credentials=%v error=%v", loaded, err)
	}
	config.Host = "old-host"
	writeTestLocator(t, appDir, config)
	oldLocator, err := LoadRemoteLocator(appDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := oldLocator.GetElevationCredentials(); err != nil {
		t.Fatal(err)
	}
	config.Host = "new-host"
	writeTestLocator(t, appDir, config)
	ForgetRemoteConfiguration(appDir)
	updatedLocator, err := LoadRemoteLocator(appDir)
	if err != nil || updatedLocator.Host != "new-host" || oldLocator.ElevationCredentials != nil {
		t.Fatalf("cached configuration survived restore: locator=%v error=%v", updatedLocator, err)
	}
	if err := os.Remove(filepath.Join(appDir, LocatorFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(appDir, ElevationCredentialsFile)); err != nil {
		t.Fatal(err)
	}
	ForgetRemoteConfiguration(appDir)
	if remote, _ := IsRemoteApp(appDir); remote {
		t.Fatal("cached locator survived removal")
	}
	if _, err := oldLocator.GetElevationCredentials(); err == nil || !strings.Contains(err.Error(), "no elevate credentials") {
		t.Fatalf("cached credentials survived removal: %v", err)
	}
}

func TestPrepareRemoteReturnsSetupFailure(t *testing.T) {
	localApp, locator, _ := newRemoteTestServer(t)
	t.Setenv("REMOTE_TEZBAKE_SOURCE", filepath.Join(t.TempDir(), "missing-binary"))
	if err := PrepareRemote(localApp, locator, "key:"+locator.PrivateKey); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preparation failure did not return its error: %v", err)
	}
}

func TestRunSshCommandReturnsMissingElevationCredentials(t *testing.T) {
	locator := &RemoteConfiguration{Elevate: REMOTE_ELEVATION_SUDO, ElevationCredentialsDirectory: t.TempDir()}
	result := runSshCommand(nil, "unused", locator, func(*ssh.Client, string, *map[string]string) *system.SshCommandResult {
		return &system.SshCommandResult{Error: errors.New("elevation required"), ExitCode: constants.ExitElevationRequired}
	})
	if result.Error == nil || result.ExitCode != -1 || !strings.Contains(result.Error.Error(), "no elevate credentials") {
		t.Fatalf("missing credentials did not return an error: %v", result)
	}
}

func TestRunSshCommandRetriesWithElevationCredentials(t *testing.T) {
	credentials := &RemoteElevateCredentials{Kind: REMOTE_ELEVATION_SUDO, Password: "literal $() password"}
	locator := &RemoteConfiguration{Elevate: REMOTE_ELEVATION_SUDO, ElevationCredentials: credentials}
	calls := 0
	result := runSshCommand(nil, "quoted command", locator, func(_ *ssh.Client, command string, env *map[string]string) *system.SshCommandResult {
		calls++
		if command != "quoted command" {
			t.Errorf("retry changed the command: %q", command)
		}
		if calls == 1 {
			if env != nil {
				t.Error("first attempt unexpectedly supplied elevation credentials")
			}
			return &system.SshCommandResult{Error: errors.New("elevation required"), ExitCode: constants.ExitElevationRequired}
		}
		if env == nil || (*env)["ELEVATION_PASSWORD"] != credentials.Password || (*env)["ELEVATION_KIND"] != "sudo" {
			t.Errorf("missing elevation credentials on retry: %v", env)
		}
		return &system.SshCommandResult{}
	})
	if calls != 2 || result.Error != nil || result.ExitCode != 0 {
		t.Fatalf("elevation retry: calls=%d result=%v", calls, result)
	}
}

func TestWriteRemoteLocatorReplacesFilesSafely(t *testing.T) {
	for _, file := range []struct {
		name string
		mode os.FileMode
	}{
		{constants.PrivateKeyFile, 0600},
		{constants.PublicKeyFile, 0644},
		{LocatorFile, 0644},
	} {
		for _, symlink := range []bool{false, true} {
			t.Run(fmt.Sprint(file.name, "/symlink=", symlink), func(t *testing.T) {
				appDir := t.TempDir()
				t.Cleanup(func() { ForgetRemoteConfiguration(appDir) })
				filePath := filepath.Join(appDir, file.name)
				outside := filepath.Join(t.TempDir(), "previous-file")
				previous := []byte("preserve outside file")
				if symlink {
					writeTestFile(t, outside, previous, 0644)
					if err := os.Symlink(outside, filePath); err != nil {
						t.Fatal(err)
					}
				} else {
					writeTestFile(t, filePath, previous, 0666)
				}
				config := &RemoteConfiguration{PrivateKey: filepath.Join(appDir, constants.PrivateKeyFile), PublicKey: filepath.Join(appDir, constants.PublicKeyFile)}
				WriteRemoteLocator(appDir, config, true)
				info, err := os.Lstat(filePath)
				if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != file.mode {
					t.Fatalf("remote configuration file was not safely replaced: info=%v error=%v", info, err)
				}
				if symlink {
					data, err := os.ReadFile(outside)
					if err != nil || string(data) != string(previous) {
						t.Fatalf("symlink target changed: data=%q error=%v", data, err)
					}
				}
			})
		}
	}
}

func TestGetAppKeyPairRepairsPublicKeyWithoutReplacingPrivateKey(t *testing.T) {
	appDir := t.TempDir()
	t.Cleanup(func() { ForgetRemoteConfiguration(appDir) })
	original := GetNewAppKeyPair()
	other := GetNewAppKeyPair()
	config := &RemoteConfiguration{PrivateKey: filepath.Join(appDir, "private.key"), PublicKey: filepath.Join(appDir, "public.key")}
	writeTestFile(t, config.PrivateKey, original.PrivateKey, 0600)
	writeTestLocator(t, appDir, config)
	for _, publicKey := range [][]byte{nil, []byte("invalid public key"), other.PublicKey, original.PublicKey} {
		if publicKey == nil {
			if err := os.Remove(config.PublicKey); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
		} else {
			writeTestFile(t, config.PublicKey, publicKey, 0644)
		}
		pair := GetAppKeyPair(appDir, false)
		if pair.IsNew || !bytes.Equal(pair.PrivateKey, original.PrivateKey) || !bytes.Equal(pair.PublicKey, original.PublicKey) {
			t.Fatal("existing private key was replaced or paired with the wrong public key")
		}
	}
	if pair := GetAppKeyPair(appDir, true); !pair.IsNew || bytes.Equal(pair.PrivateKey, original.PrivateKey) {
		t.Fatal("explicit rekey reused the existing private key")
	}
}

func TestWriteAppDefinitionDoesNotFollowTemporarySymlink(t *testing.T) {
	appDir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "preserve")
	writeTestFile(t, outside, []byte("outside file"), 0644)
	if err := os.Symlink(outside, filepath.Join(appDir, "app.json.new")); err != nil {
		t.Fatal(err)
	}
	if err := WriteAppDefinition(appDir, map[string]any{"user": "new-user"}, "app.json"); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "outside file" {
		t.Fatalf("configuration write followed a temporary symlink: data=%q error=%v", data, err)
	}
	definition, err := ReadAppDefinition(appDir, "app.json")
	if err != nil || definition["user"] != "new-user" {
		t.Fatalf("configuration was not saved: definition=%v error=%v", definition, err)
	}
}
