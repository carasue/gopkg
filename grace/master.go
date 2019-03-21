// +build linux darwin netbsd freebsd openbsd dragonfly

package grace

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type master struct {
	sync.Mutex

	listeners []*os.File
	worker    *os.Process
	logger    logger
}

func (p *master) run(addrs []string) {
	if p.logger == nil {
		p.logger = &stdLogger{}
	}
	for _, addr := range addrs {
		parts := strings.SplitN(addr, "://", 2)
		lnet, laddr := parts[0], parts[1]
		l, err := net.Listen(lnet, laddr)
		if err != nil {
			p.logger.Fatalf("master: listen address %s: %v", addr, err)
		}
		file, err := l.(interface{ File() (*os.File, error) }).File()
		if err != nil {
			p.logger.Fatalf("master: get listener file %s: %v", addr, err)
		}
		p.listeners = append(p.listeners, file)
		if err = l.Close(); err != nil {
			p.logger.Fatalf("master: close listner %s: %v", addr, err)
		}
	}

	// fork the initial worker
	proc, err := p.forkWorker()
	if err != nil {
		p.logger.Fatalf("master: fork initial worker: %v", err)
	}
	p.worker = proc

	// watch signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals)
	for s := range signals {
		p.handleSignal(s)
	}
}

func (p *master) handleSignal(s os.Signal) {
	if s.String() == "child exited" {
		return
	}
	p.Lock()
	defer p.Unlock()

	if s == syscall.SIGUSR1 {
		proc, err := p.forkWorker()
		if err != nil {
			p.logger.Errorf("master: fork new worker: %v", err)
			return
		}
		oldWorker := p.worker
		p.worker = proc
		// signal the old worker to stop accept connections and shutdown
		oldWorker.Signal(syscall.SIGHUP)
	} else {
		// proxy other signals to the active worker process
		err := p.worker.Signal(s)
		if err != nil {
			p.logger.Errorf("master: signal worker process: %v", err)
		}
	}
}

func (p *master) forkWorker() (proc *os.Process, err error) {
	os.Setenv(EnvInheritCnt, strconv.Itoa(len(p.listeners)))
	execSpec := &os.ProcAttr{
		Env:   os.Environ(),
		Files: append([]*os.File{os.Stdin, os.Stdout, os.Stderr}, p.listeners...),
	}
	proc, err = os.StartProcess(os.Args[0], os.Args, execSpec)
	if err != nil {
		return nil, err
	}
	if err = p.waitWorker(proc); err != nil {
		return nil, err
	}
	return
}

func (p *master) waitWorker(proc *os.Process) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer cancel()
		ps, err := proc.Wait()
		for err != nil {
			p.logger.Errorf("master: wait worker process %d: %v", proc.Pid, err)
			time.Sleep(time.Second)
			ps, err = proc.Wait()
		}
		if ps.Success() {
			p.logger.Infof("master: process %d exited gracefully", proc.Pid)
		} else {
			p.logger.Errorf("master: process %d exited abnormally: %v", proc.Pid, ps.String())
		}

		// stop master process if the active worker stopped
		p.Lock()
		defer p.Unlock()
		isActiveWorker := p.worker == proc
		if isActiveWorker {
			if ps.Success() {
				os.Exit(0)
			} else {
				os.Exit(1)
			}
		}
	}()

	// wait the worker process to be ready
	select {
	case <-ctx.Done():
		return errors.New("worker process exit quickly")
	case <-time.After(3 * time.Second):
		return nil
	}
}
