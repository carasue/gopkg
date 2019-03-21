// +build !linux,!darwin,!netbsd,!freebsd,!openbsd,!dragonfly

package grace

import "net"

func Run(addrs []string, main func(), logger logger) {
	main()
}

func Listen(network string, address string) (net.Listener, error) {
	return net.Listen(network, address)
}
