// Package client provides a test-infrastructure helper
// that Go integration tests use to orchestrate the
// it_sidecar process from TestMain.
package client

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Callback is invoked after services are ready but
// before tests run.
type Callback func() error

// K8STestSetup orchestrates the it_sidecar process
// given the pods and services we must wait for.
type K8STestSetup struct {
	WaitForPods         []string
	PortForwardServices map[string]int
	ReadyCallback       Callback

	forwards map[string]int
	cmd      *exec.Cmd
	in       io.WriteCloser
	out      io.ReadCloser
	er       io.ReadCloser
}

//nolint:gochecknoglobals // test infra flag
var setupCMD = flag.String(
	"setup", "",
	"the path to the it setup command",
)

// TestMain executes the sidecar, waits for pods and
// services, runs the ReadyCallback if set, then
// executes the test suite. On completion it signals
// the sidecar to tear down.
func (s *K8STestSetup) TestMain(m *testing.M) {
	s.forwards = make(map[string]int)

	wg := new(sync.WaitGroup)
	wg.Add(2) //nolint:mnd // stdout + stderr

	os.Exit(func() int {
		flag.Parse()

		defer func() {
			// Closing stdin signals the sidecar to
			// exit.
			//nolint:errcheck,gosec // best-effort close
			s.in.Close()
			wg.Wait()

			if err := s.cmd.Wait(); err != nil {
				log.Fatal(err)
			}
		}()

		s.before(wg)

		if s.ReadyCallback != nil {
			if err := s.ReadyCallback(); err != nil {
				log.Fatal(err)
			}
		}

		return m.Run()
	}())
}

// GetServiceLocalPort returns the locally-forwarded
// port for the given service name.
func (s *K8STestSetup) GetServiceLocalPort(
	serviceName string,
) int {
	return s.forwards[serviceName]
}

func (s *K8STestSetup) before(wg *sync.WaitGroup) {
	log.Printf("setup command: %s\n", *setupCMD)

	args := make([]string, 0, len(s.WaitForPods)+
		len(s.PortForwardServices))

	for _, app := range s.WaitForPods {
		args = append(
			args,
			fmt.Sprintf("-waitforapp=%s", app),
		)
	}

	for service, port := range s.PortForwardServices {
		args = append(
			args,
			fmt.Sprintf(
				"-portforward=%s:%d", service, port,
			),
		)
	}

	//nolint:gosec,noctx // test infra command from flag
	s.cmd = exec.Command(*setupCMD, args...)

	var err error

	// Open stderr pipe. StderrPipe is closed
	// automatically by Wait.
	if s.er, err = s.cmd.StderrPipe(); err != nil {
		log.Fatal(
			fmt.Errorf(
				"unable to read setup STDERR: %w",
				err,
			),
		)
	}

	go func() {
		rd := bufio.NewReader(s.er)

		for {
			str, readErr := rd.ReadString('\n')
			if readErr == io.EOF {
				break
			}

			if readErr != nil {
				log.Fatal(readErr)
			}

			log.Print(str)
		}

		wg.Done()
	}()

	if s.out, err = s.cmd.StdoutPipe(); err != nil {
		log.Fatal(err)
	}

	if s.in, err = s.cmd.StdinPipe(); err != nil {
		log.Fatal(err)
	}

	if err = s.cmd.Start(); err != nil {
		log.Fatal(err)
	}

	rd := bufio.NewReader(s.out)

	for {
		str, readErr := rd.ReadString('\n')
		if readErr != nil {
			log.Fatal(
				"unable to read from setup stdout",
			)
		}

		fmt.Print(str)

		if strings.HasPrefix(str, "FORWARD") {
			parts := strings.Split(
				strings.TrimSpace(str[8:]), ":",
			)

			const expectedParts = 3
			if len(parts) == expectedParts {
				localPort, atoiErr := strconv.Atoi(
					parts[2],
				)
				if atoiErr != nil {
					log.Printf(
						"invalid port %q: %v",
						parts[2], atoiErr,
					)

					continue
				}

				s.forwards[parts[0]] = localPort
			}
		}

		if str == "READY\n" {
			break
		}
	}

	go func() {
		for {
			str, readErr := rd.ReadString('\n')
			if readErr == io.EOF {
				break
			}

			if readErr != nil {
				log.Fatal(readErr)
			}

			log.Print(str)
		}

		wg.Done()
	}()
}
