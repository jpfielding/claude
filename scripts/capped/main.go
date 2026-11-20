// capped runs a command in its own process group with a hard RSS cap and a
// wall-clock timeout, killing the whole group (SIGKILL) if either is exceeded.
// RLIMIT_AS is also set (enforced on Linux; macOS ignores it, hence the RSS
// watchdog, which is the real guard there).
//
//	capped -mem 4096 -timeout 10s -- cmd args...
//
// Exit codes: child's code; 137 = killed for memory; 124 = timeout.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	memMB := flag.Int64("mem", 4096, "RSS cap for the process group, MiB")
	timeout := flag.Duration("timeout", 10*time.Second, "wall-clock limit")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: capped -mem MiB -timeout D -- cmd args...")
		os.Exit(2)
	}
	limit := uint64(*memMB) << 20
	_ = syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{Cur: limit, Max: limit}) // inherited; Linux only in practice

	cmd := exec.Command(flag.Arg(0), flag.Args()[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOMEMLIMIT=%dMiB", *memMB/2))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "capped:", err)
		os.Exit(2)
	}
	pgid := cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	deadline := time.After(*timeout)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	kill := func(code int, why string) {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		<-done
		fmt.Fprintf(os.Stderr, "capped: KILLED (%s): %s\n", why, strings.Join(flag.Args(), " "))
		os.Exit(code)
	}
	for {
		select {
		case err := <-done:
			if ee, ok := err.(*exec.ExitError); ok {
				os.Exit(ee.ExitCode())
			}
			if err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		case <-deadline:
			kill(124, "timeout "+timeout.String())
		case <-tick.C:
			if rss := groupRSS(pgid); rss > limit {
				kill(137, fmt.Sprintf("rss %d MiB > %d MiB", rss>>20, *memMB))
			}
		}
	}
}

// groupRSS sums resident memory (bytes) of every process in pgid.
func groupRSS(pgid int) uint64 {
	out, err := exec.Command("ps", "-A", "-o", "pgid=,rss=").Output()
	if err != nil {
		return 0
	}
	var total uint64
	for _, line := range bytes.Split(out, []byte("\n")) {
		f := strings.Fields(string(line))
		if len(f) != 2 || f[0] != strconv.Itoa(pgid) {
			continue
		}
		kb, _ := strconv.ParseUint(f[1], 10, 64)
		total += kb << 10
	}
	return total
}
