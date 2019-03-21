// +build linux darwin netbsd freebsd openbsd dragonfly

package grace

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

const EnvInheritCnt = "_INHERIT_FDS"

var state = struct {
	inheritOnce sync.Once

	sync.Mutex
	inherited []net.Listener
	active    []net.Listener
}{}

func Run(addrs []string, main func(), logger logger) {
	if os.Getenv(EnvInheritCnt) != "" { // worker
		main()
	} else { // master
		mp := &master{logger: logger}
		mp.run(addrs)
	}
}

func Listen(network string, address string) (net.Listener, error) {
	if err := inherit(); err != nil {
		return nil, err
	}
	s := &state
	s.Lock()
	defer s.Unlock()

	// look for an inherited listener
	for i, l := range s.inherited {
		if l != nil && isSameAddr(l.Addr(), network, address) {
			s.inherited[i] = nil
			s.active = append(s.active, l)
			return l.(net.Listener), nil
		}
	}

	// make a fresh listener
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	s.active = append(s.active, l)
	return l, nil
}

func inherit() error {
	var s = &state
	var retErr error
	s.inheritOnce.Do(func() {
		s.Lock()
		defer s.Unlock()

		countStr := os.Getenv(EnvInheritCnt)
		if countStr == "" {
			return
		}
		count, err := strconv.Atoi(countStr)
		if err != nil {
			retErr = fmt.Errorf("invalid count value: %s=%s", EnvInheritCnt, countStr)
			return
		}
		fdStart := 3
		for i := fdStart; i < fdStart+count; i++ {
			file := os.NewFile(uintptr(i), "listener")
			l, err := net.FileListener(file)
			if err != nil {
				file.Close()
				retErr = fmt.Errorf("inheriting socket fd %d: %v", i, err)
				return
			}
			if err := file.Close(); err != nil {
				retErr = fmt.Errorf("closing inherited socket fd %d: %v", i, err)
				return
			}
			s.inherited = append(s.inherited, l)
		}
	})
	return retErr
}

func isSameAddr(a1 net.Addr, lnet, laddr string) bool {
	if a1.Network() != lnet {
		return false
	}
	var a2 net.Addr
	var err error
	switch lnet {
	case "tcp", "tcp4", "tcp6":
		a2, err = net.ResolveTCPAddr(lnet, laddr)
	case "unix", "unixpacket":
		a2, err = net.ResolveUnixAddr(lnet, laddr)
	}
	if err != nil || a2 == nil {
		return false
	}

	a1s := a1.String()
	a2s := a2.String()
	if a1s == a2s {
		return true
	}

	// This allows for ipv6 vs ipv4 local addresses to compare as equal. This
	// scenario is common when listening on localhost.
	const ipv6prefix = "[::]"
	const ipv4prefix = "0.0.0.0"
	a1s = strings.TrimPrefix(a1s, ipv6prefix)
	a2s = strings.TrimPrefix(a2s, ipv6prefix)
	a1s = strings.TrimPrefix(a1s, ipv4prefix)
	a2s = strings.TrimPrefix(a2s, ipv4prefix)
	return a1s == a2s
}
