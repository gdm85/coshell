package cosh

import (
	"errors"
	"fmt"

	"syscall"
)

// POSIX signals as listed in RFC 4254 Section 6.10.
// Copied from https://github.com/golang/crypto/blob/master/ssh/session.go
const (
	SIGABRT = "ABRT"
	SIGALRM = "ALRM"
	SIGFPE  = "FPE"
	SIGHUP  = "HUP"
	SIGILL  = "ILL"
	SIGINT  = "INT"
	SIGKILL = "KILL"
	SIGPIPE = "PIPE"
	SIGQUIT = "QUIT"
	SIGSEGV = "SEGV"
	SIGTERM = "TERM"
	SIGUSR1 = "USR1"
	SIGUSR2 = "USR2"
)

var signals = map[string]syscall.Signal{
	SIGABRT: syscall.SIGABRT,
	SIGALRM: syscall.SIGALRM,
	SIGFPE:  syscall.SIGFPE,
	SIGHUP:  syscall.SIGHUP,
	SIGILL:  syscall.SIGILL,
	SIGINT:  syscall.SIGINT,
	SIGKILL: syscall.SIGKILL,
	SIGPIPE: syscall.SIGPIPE,
	SIGQUIT: syscall.SIGQUIT,
	SIGSEGV: syscall.SIGSEGV,
	SIGTERM: syscall.SIGTERM,
	SIGUSR1: syscall.SIGUSR1,
	SIGUSR2: syscall.SIGUSR2,
}

func ParseSignal(s string) (syscall.Signal, error) {
	switch s {
	case SIGABRT, fmt.Sprintf("%d", signals[SIGABRT]):
		return signals[SIGABRT], nil
	case SIGALRM, fmt.Sprintf("%d", signals[SIGALRM]):
		return signals[SIGABRT], nil
	case SIGFPE, fmt.Sprintf("%d", signals[SIGFPE]):
		return signals[SIGABRT], nil
	case SIGHUP, fmt.Sprintf("%d", signals[SIGHUP]):
		return signals[SIGABRT], nil
	case SIGILL, fmt.Sprintf("%d", signals[SIGILL]):
		return signals[SIGABRT], nil
	case SIGINT, fmt.Sprintf("%d", signals[SIGINT]):
		return signals[SIGABRT], nil
	case SIGKILL, fmt.Sprintf("%d", signals[SIGKILL]):
		return signals[SIGABRT], nil
	case SIGPIPE, fmt.Sprintf("%d", signals[SIGPIPE]):
		return signals[SIGABRT], nil
	case SIGQUIT, fmt.Sprintf("%d", signals[SIGQUIT]):
		return signals[SIGABRT], nil
	case SIGSEGV, fmt.Sprintf("%d", signals[SIGSEGV]):
		return signals[SIGABRT], nil
	case SIGTERM, fmt.Sprintf("%d", signals[SIGTERM]):
		return signals[SIGABRT], nil
	case SIGUSR1, fmt.Sprintf("%d", signals[SIGUSR1]):
		return signals[SIGABRT], nil
	case SIGUSR2, fmt.Sprintf("%d", signals[SIGUSR2]):
		return signals[SIGABRT], nil
	}

	return -1, errors.New("unknown signal")
}
