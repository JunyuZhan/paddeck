package collector

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSH runs commands on one remote host with key authentication.
type SSH struct {
	Addr    string
	User    string
	Port    int
	KeyPath string
	Timeout time.Duration
}

func (s *SSH) networkAddr() string {
	port := s.Port
	if port == 0 {
		port = 22
	}
	return net.JoinHostPort(s.Addr, fmt.Sprint(port))
}

// expandHome turns "~/x" into an absolute path.
func expandHome(p string) string {
	if !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, p[2:])
}

func loadSigner(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		return nil, fmt.Errorf("read ssh key %s: %w", path, err)
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse ssh key %s: %w", path, err)
	}
	return signer, nil
}

// authMethods picks credentials without forcing the user to configure a key:
// an explicit key, then the usual ~/.ssh keys, then ssh-agent.
func (s *SSH) authMethods() ([]ssh.AuthMethod, error) {
	if s.KeyPath != "" {
		signer, err := loadSigner(s.KeyPath)
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{"id_ed25519", "id_rsa", "id_ecdsa"}
	for _, name := range candidates {
		p := filepath.Join(home, ".ssh", name)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if signer, err := loadSigner(p); err == nil {
			return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
		}
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			return []ssh.AuthMethod{ssh.PublicKeysCallback(agent.NewClient(conn).Signers)}, nil
		}
	}
	return nil, fmt.Errorf("no SSH credentials: set collector.ssh_key, " +
		"or create ~/.ssh/id_ed25519, or start ssh-agent")
}

func (s *SSH) clientConfig() (*ssh.ClientConfig, error) {
	auth, err := s.authMethods()
	if err != nil {
		return nil, err
	}
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 8 * time.Second
	}
	return &ssh.ClientConfig{
		User:            s.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}, nil
}

// Run executes a shell script and returns stdout. stderr is appended on error.
func (s *SSH) Run(ctx context.Context, script string) (string, error) {
	cfg, err := s.clientConfig()
	if err != nil {
		return "", err
	}
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 8 * time.Second
	}

	type result struct {
		out string
		err error
	}
	ch := make(chan result, 1)

	go func() {
		dialer := net.Dialer{Timeout: timeout}
		conn, err := dialer.DialContext(ctx, "tcp", s.networkAddr())
		if err != nil {
			ch <- result{err: fmt.Errorf("dial %s: %w", s.networkAddr(), err)}
			return
		}
		sshConn, chans, reqs, err := ssh.NewClientConn(conn, s.networkAddr(), cfg)
		if err != nil {
			conn.Close()
			ch <- result{err: fmt.Errorf("ssh handshake: %w", err)}
			return
		}
		client := ssh.NewClient(sshConn, chans, reqs)
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			ch <- result{err: fmt.Errorf("ssh session: %w", err)}
			return
		}
		defer session.Close()

		var stdout, stderr bytes.Buffer
		session.Stdout = &stdout
		session.Stderr = &stderr
		if err := session.Run(script); err != nil {
			msg := stderr.String()
			if msg == "" {
				msg = err.Error()
			}
			ch <- result{err: fmt.Errorf("remote command: %s", msg)}
			return
		}
		ch <- result{out: stdout.String()}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		return r.out, r.err
	}
}
