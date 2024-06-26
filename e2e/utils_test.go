package e2e

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/onsi/gomega"
)

const (
	jqPath           = "../bin/download/jq"
	kubectlPath      = "../bin/download/kubectl"
	policyViewerPath = "/tmp/cilium-policy"
)

func runCommand(path string, input []byte, args ...string) ([]byte, []byte, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := exec.Command(path, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), stderr.Bytes(), nil
	}
	_, file := filepath.Split(path)
	return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s failed with %s: stderr=%s", file, err, stderr)
}

func kubectl(input []byte, args ...string) ([]byte, []byte, error) {
	return runCommand(kubectlPath, input, args...)
}

func kubectlSafe(g Gomega, input []byte, args ...string) []byte {
	stdout, stderr, err := runCommand(kubectlPath, input, args...)
	g.Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)
	return stdout
}

func jqSafe(g Gomega, input []byte, args ...string) []byte {
	stdout, stderr, err := runCommand(jqPath, input, args...)
	g.Expect(err).NotTo(HaveOccurred(), "input: %s, stdout: %s, stderr: %s", string(input), stdout, stderr)
	return []byte(strings.TrimSpace(string(stdout)))
}

func runViewer(input []byte, args ...string) ([]byte, []byte, error) {
	args = append([]string{"exec", "deploy/ubuntu", "--", policyViewerPath}, args...)
	return kubectl(input, args...)
}

func runViewerSafe(g Gomega, input []byte, args ...string) []byte {
	stdout, stderr, err := runViewer(input, args...)
	g.Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)
	return stdout
}

func onePodByLabelSelector(g Gomega, namespace, selector string) string {
	data := kubectlSafe(g, nil, "get", "pod", "-n", namespace, "-l", selector, "-o=json")
	count, err := strconv.Atoi(string(jqSafe(g, data, "-r", ".items | length")))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(count).To(Equal(1))
	return string(jqSafe(g, data, "-r", ".items[0].metadata.name"))
}
