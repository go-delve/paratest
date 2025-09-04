package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <test-binary> [args...]\n", os.Args[0])
	}

	testBinary := os.Args[1]
	testArgs := os.Args[2:]

	// Get all tests from the binary
	tests, err := getTestList(testBinary)
	if err != nil {
		log.Fatalf("Error getting test list: %v\n", err)
	}

	// Check if we should parallelize (>20 tests).
	// TODO(derekparker): make this configurable.
	if len(tests) <= 20 {
		// Just run the test binary normally
		runTestBinary(testBinary, testArgs)
		return
	}

	n := runtime.GOMAXPROCS(0)
	testGroups := divideTests(tests, n)

	runTestsInParallel(testBinary, testArgs, testGroups)
}

// getTestList gets all test function names from the test binary
func getTestList(testBinary string) ([]string, error) {
	cmd := exec.Command(testBinary, "-test.list", ".*")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list tests: %v", err)
	}

	var tests []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && strings.HasPrefix(line, "Test") {
			tests = append(tests, line)
		}
	}

	return tests, scanner.Err()
}

// divideTests divides tests into n groups, putting remainder in the last group
func divideTests(tests []string, n int) [][]string {
	if n <= 0 {
		n = 1
	}
	if len(tests) == 0 {
		groups := make([][]string, n)
		for i := range groups {
			groups[i] = []string{}
		}
		return groups
	}
	if n > len(tests) {
		n = len(tests)
	}

	groupSize := len(tests) / n
	remainder := len(tests) % n

	groups := make([][]string, n)
	index := 0

	for i := 0; i < n; i++ {
		size := groupSize
		if i == n-1 {
			size += remainder // add remainder to last group
		}

		groups[i] = make([]string, size)
		for j := 0; j < size && index < len(tests); j++ {
			groups[i][j] = tests[index]
			index++
		}
	}

	return groups
}

// runTestsInParallel runs test groups in parallel
func runTestsInParallel(testBinary string, testArgs []string, testGroups [][]string) {
	var wg sync.WaitGroup
	var exitCode atomic.Int64

	for _, group := range testGroups {
		if len(group) == 0 {
			continue
		}

		wg.Go(func() {
			// Create regex pattern for this group
			pattern := "^(" + strings.Join(group, "|") + ")$"

			args := append([]string{"-test.run", pattern}, testArgs...)

			cmd := exec.Command(testBinary, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				exitCode.Store(1)

				if exitErr, ok := err.(*exec.ExitError); ok {
					if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
						exitCode.Store(int64(status.ExitStatus()))
					}
				}
			}
		})
	}

	wg.Wait()

	code := exitCode.Load()
	if code != 0 {
		os.Exit(int(code))
	}
}

// runTestBinary runs the test binary with given arguments (fallback case)
func runTestBinary(testBinary string, args []string) {
	cmd := exec.Command(testBinary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		os.Exit(1)
	}
}
