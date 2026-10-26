package main

import (
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
)

func run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func connectSSH(user string, host string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password("root"),
		},
	}

	return ssh.Dial("tcp", net.JoinHostPort(host, "22"), config)
}
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func runSSH(client *ssh.Client, dir string, name string, args ...string) error {
	return runSSHExplicit(client, dir, nil, nil, nil, name, args...)
}
func runSSHExplicit(client *ssh.Client, dir string, stdin *os.File, stdout *os.File, stderr *os.File, name string, args ...string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if stdin != nil {
		session.Stdin = stdin
	}
	if stdout != nil {
		session.Stdout = stdout
	} else {
		session.Stdout = io.Discard
	}
	if stderr != nil {
		session.Stderr = stderr
	} else {
		session.Stderr = io.Discard
	}

	// Build the command
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(name))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	cmd := strings.Join(parts, " ")
	// Move to the right directory
	if dir != "" {
		cmd = "cd " + shellQuote(dir) + " && " + cmd
	}

	return session.Run(cmd)
}
