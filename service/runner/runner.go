package runner

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/z46-dev/freeipa-runner/config"
)

type BashResponse struct {
	Host     string
	Response string // stdout + stderr (combined)
	Error    error
}

func RunBashScript(bashScriptFile string, hosts []string) (responses []BashResponse, err error) {
	return runScriptGeneric(bashScriptFile, hosts, "/bin/bash", "sh")
}

func RunPythonScript(pythonScriptFile string, hosts []string) (responses []BashResponse, err error) {
	return runScriptGeneric(pythonScriptFile, hosts, "/usr/bin/python3", "py")
}

// NOTE: This shells out to `ansible-playbook` with a temp inventory
func RunAnsiblePlaybook(playbookFile string, hosts []string) (responses []BashResponse, err error) {
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts provided")
	}

	tmpDir, err := os.MkdirTemp("", "fir-ansible-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	inventory := "[targets]\n" + strings.Join(hosts, "\n") + "\n"
	invPath := filepath.Join(tmpDir, "inventory.ini")
	if err := os.WriteFile(invPath, []byte(inventory), 0644); err != nil {
		return nil, err
	}

	args := []string{"-i", invPath, playbookFile, "-l", "targets", "-b"}
	// Kerberos-friendly env (ansible already supports GSSAPI via ssh config)
	cmd := exec.CommandContext(contextWithTimeout(), "ansible-playbook", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	runErr := cmd.Run()
	responses = append(responses, BashResponse{
		Host:     "(ansible)",
		Response: out.String(),
		Error:    runErr,
	})
	return responses, nil
}

// -------- internals

func contextWithTimeout() context.Context {
	secs := time.Duration(config.Config.SSH.TimeoutSeconds) * time.Second
	ctx, _ := context.WithTimeout(context.Background(), secs)
	return ctx
}

func randSuffix(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func runScriptGeneric(localPath string, hosts []string, interp string, ext string) ([]BashResponse, error) {
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts provided")
	}
	if _, err := os.Stat(localPath); err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	concurrency := config.Config.SSH.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	type job struct {
		host string
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []BashResponse

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			resp := runOnce(interp, ext, localPath, j.host)
			mu.Lock()
			results = append(results, resp)
			mu.Unlock()
		}
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker()
	}
	for _, h := range hosts {
		jobs <- job{host: h}
	}
	close(jobs)
	wg.Wait()

	return results, nil
}

func runOnce(interp, ext, localPath, host string) BashResponse {
	ctx := contextWithTimeout()

	// Remote staging
	tag := randSuffix(4)
	remoteDir := fmt.Sprintf("/tmp/freeipa-runner-%s", tag)
	remoteFile := fmt.Sprintf("%s/script.%s", remoteDir, ext)
	unit := fmt.Sprintf("%s-%s", config.Config.SSH.SystemdUnitPrefix, tag)

	sshUser := config.Config.SSH.User
	dest := fmt.Sprintf("%s@%s", sshUser, host)

	sshBase := []string{"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes"}
	if kp := strings.TrimSpace(config.Config.SSH.KnownHostsPath); kp != "" {
		sshBase = append(sshBase, "-o", "UserKnownHostsFile="+kp)
	}

	// If using Kerberos, we prefer system ssh config to handle GSSAPI; otherwise allow key auth.
	if !config.Config.SSH.UseKerberos && strings.TrimSpace(config.Config.SSH.PrivateKeyPath) != "" {
		sshBase = append(sshBase, "-i", config.Config.SSH.PrivateKeyPath)
	}

	// 1) mkdir on remote
	if out, err := runCmd(ctx, "ssh", append(sshBase, dest, "mkdir", "-p", remoteDir)...); err != nil {
		return BashResponse{Host: host, Response: out, Error: fmt.Errorf("mkdir failed: %w", err)}
	}

	// 2) scp script
	if out, err := runCmd(ctx, "scp", append(sshBase, localPath, dest+":"+remoteFile)...); err != nil {
		return BashResponse{Host: host, Response: out, Error: fmt.Errorf("scp failed: %w", err)}
	}

	// 3) chmod +x
	if out, err := runCmd(ctx, "ssh", append(sshBase, dest, "chmod", "+x", remoteFile)...); err != nil {
		return BashResponse{Host: host, Response: out, Error: fmt.Errorf("chmod failed: %w", err)}
	}

	// 4) systemd-run (optionally sudo)
	prefix := []string{"ssh"}
	prefix = append(prefix, sshBase...)
	prefix = append(prefix, dest)

	sudo := ""
	if config.Config.SSH.Sudo {
		sudo = "sudo"
	}

	runCmdline := fmt.Sprintf("%s /usr/bin/systemd-run --collect --wait --unit %s %s %s",
		sudo, unit, interp, remoteFile)

	out, err := runCmd(ctx, prefix[0], append(prefix[1:], runCmdline)...)
	if err != nil {
		return BashResponse{Host: host, Response: out, Error: fmt.Errorf("systemd-run failed: %w", err)}
	}

	// Optional: journal dump (to ensure we collected full logs)
	journalCmd := fmt.Sprintf("%s /usr/bin/journalctl -u %s -n 500 --no-pager", sudo, unit)
	jout, _ := runCmd(ctx, prefix[0], append(prefix[1:], journalCmd)...)

	combined := strings.TrimSpace(out + "\n---\n" + jout)
	return BashResponse{Host: host, Response: combined, Error: nil}
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
