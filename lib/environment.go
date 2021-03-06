package vaulted

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Environment struct {
	Expiration int64             `json:"expiration"`
	Vars       map[string]string `json:"vars"`
	SSHKeys    map[string]string `json:"ssh_keys,omitempty"`
}

func (e *Environment) Spawn(cmd []string, extraVars map[string]string) (*int, error) {
	if len(cmd) == 0 {
		return nil, ErrInvalidCommand
	}

	// lookup the path of the executable
	cmdpath, err := exec.LookPath(cmd[0])
	if err != nil {
		return nil, fmt.Errorf("Cannot find executable %s: %v", cmd[0], err)
	}

	// copy the extra vars so we can mutate it
	vars := make(map[string]string)
	for key, value := range extraVars {
		vars[key] = value
	}

	// start the agent
	sock, err := e.startProxyKeyring()
	if err != nil {
		return nil, err
	}

	vars["SSH_AUTH_SOCK"] = sock

	// start the process
	var attr os.ProcAttr
	attr.Env = e.buildEnviron(vars)
	attr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}

	proc, err := os.StartProcess(cmdpath, cmd, &attr)
	if err != nil {
		return nil, fmt.Errorf("Failed to execute command: %v", err)
	}

	// wait for the process to exit
	state, _ := proc.Wait()

	var exitStatus int
	if !state.Success() {
		if status, ok := state.Sys().(syscall.WaitStatus); ok {
			exitStatus = status.ExitStatus()
		} else {
			exitStatus = 255
		}
	}

	// we only return an error if spawning the process failed, not if
	// the spawned command returned a failure status code.
	return &exitStatus, nil
}

func (e *Environment) startProxyKeyring() (string, error) {
	keyring, err := NewProxyKeyring(os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return "", err
	}

	// load ssh keys
	for comment, key := range e.SSHKeys {
		addedKey := agent.AddedKey{
			Comment: comment,
		}

		addedKey.PrivateKey, err = ssh.ParseRawPrivateKey([]byte(key))
		if err != nil {
			return "", err
		}

		err := keyring.Add(addedKey)
		if err != nil {
			return "", err
		}
	}

	sock, err := keyring.Listen()
	if err != nil {
		return "", err
	}

	go keyring.Serve()

	return sock, err
}

func (e *Environment) buildEnviron(extraVars map[string]string) []string {
	// load the current environ
	env := make(map[string]string)
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		env[parts[0]] = parts[1]
	}

	// merge the vars
	for key, value := range e.Vars {
		env[key] = value
	}
	for key, value := range extraVars {
		env[key] = value
	}

	// recombine into environ
	environ := make([]string, 0, len(env))
	for key, value := range env {
		environ = append(environ, fmt.Sprintf("%s=%s", key, value))
	}
	return environ
}
