//go:build (!linux && !darwin && !freebsd && !windows) && cmfa_smart

package tcpstats

import "syscall"

func readTCPStats(rawConn syscall.RawConn) *Stats {
	return nil
}
