//go:build linux
// +build linux

package tcp

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"golang.org/x/net/proxy"
)

// Proxy forwards a TCP request to a TCP service.
type ProxyBPF struct {
	address       string
	proxyProtocol *dynamic.ProxyProtocol
	dialer        proxy.Dialer
	hashmap       *ebpf.Map
}

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cflags "-Wall -Wextra -g -O2" bpf proxy-sockmap.c -- -I./include

// NewProxy creates a new Proxy.
func NewProxyBPF(ctx context.Context, address string, proxyProtocol *dynamic.ProxyProtocol, dialer proxy.Dialer) (*ProxyBPF, error) {
	if proxyProtocol != nil && (proxyProtocol.Version < 1 || proxyProtocol.Version > 2) {
		return nil, fmt.Errorf("unknown proxyProtocol version: %d", proxyProtocol.Version)
	}

	objs := bpfObjects{}

	if err := loadBpfObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("loading objects: %w", err)
	}
	go func() {
		<-ctx.Done()
		objs.Close()
	}()

	var err error

	err = link.RawAttachProgram(link.RawAttachProgramOptions{
		Target:  objs.HashMap.FD(),
		Program: objs.ProgVerdict,
		Attach:  ebpf.AttachSkSKBStreamVerdict,
	})
	if err != nil {
		return nil, err
	}

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, err
	}

	return &ProxyBPF{
		hashmap:       objs.HashMap,
		address:       address,
		proxyProtocol: proxyProtocol,
		dialer:        dialer,
	}, nil
}

// ServeTCP forwards the connection to a service.
func (p *ProxyBPF) ServeTCP(conn WriteCloser) {
	log.Debug().
		Str("address", p.address).
		Str("remoteAddr", conn.RemoteAddr().String()).
		Msg("Handling connection")

	// needed because of e.g. server.trackedConnection
	defer conn.Close()

	connBackend, err := p.dialBackend()
	if err != nil {
		log.Error().Err(err).Msg("Error while connecting to backend")
		return
	}

	log.Debug().
		Msg("dial backend done")
	// maybe not needed, but just in case
	defer connBackend.Close()

	b := make([]byte, 4096)
	n, err := conn.Read(b)
	if err != nil {
		log.Error().Err(err).Msg("Error while reading first bytes of the connection")
		return
	}
	log.Debug().
		Msg("first read done")
	_, err = connBackend.Write(b[:n])
	if err != nil {
		log.Error().Err(err).Msg("Error while writing first bytes in the backend connection")
		return
	}
	log.Debug().
		Msg("first write done")

	key, err := getKey(connBackend)
	if err != nil {
		log.Error().Err(err).Msg("Error while calculating backend key")
		return
	}
	log.Debug().
		Msg("getKey backend done")

	sconn, ok := conn.(syscall.Conn)
	if !ok {
		log.Error().Msgf("Unable to get the file descriptor for the client conn %T", conn)
		return
	}

	rawConn, err := sconn.SyscallConn()
	if err != nil {
		log.Error().Err(err).Msg("Error while taking file descriptor")
		return
	}
	rawConn.Control(func(fd uintptr) {
		err = p.hashmap.Update(key, uint32(fd), ebpf.UpdateAny)
		if err != nil {
			log.Error().Uint32("fd", uint32(fd)).Err(err).Msg("Error while taking file descriptor")
			return
		}
	})
	log.Debug().
		Msg("hash update 1 done")

	u, err := getKey(conn)
	if err != nil {
		log.Error().Err(err).Msg("Error while calculating client key")
		return
	}

	sconnBackend, ok := connBackend.(syscall.Conn)
	if !ok {
		log.Error().Msgf("Unable to get the file descriptor for the backend conn %T", connBackend)
		return
	}

	rawConnBackend, err := sconnBackend.SyscallConn()
	if err != nil {
		log.Error().Err(err).Msg("Error while taking file descriptor")
		return
	}
	rawConnBackend.Control(func(ofd uintptr) {
		err = p.hashmap.Update(u, uint32(ofd), ebpf.UpdateAny)
		if err != nil {
			log.Error().Uint32("fd", uint32(ofd)).Err(err).Msg("Error while taking file descriptor")
			return
		}
	})

	log.Debug().
		Msg("hash update 2 done")

	// This will not read anything, but it will block until conn is closed
	conn.Read(b)

	p.hashmap.Delete(u)
	p.hashmap.Delete(key)

}

func (p *ProxyBPF) dialBackend() (WriteCloser, error) {
	conn, err := p.dialer.Dial("tcp", p.address)
	if err != nil {
		return nil, err
	}

	return conn.(WriteCloser), nil
}

func getFD(conn syscall.Conn) (uintptr, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var connfd uintptr
	err = rawConn.Control(func(fd uintptr) { connfd = fd })
	if err != nil {
		return 0, err
	}
	return connfd, nil
}

func getKey(conn net.Conn) (uint64, error) {
	_, local_port, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return 0, err
	}
	_, remote_port, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return 0, err
	}

	ilocal_port, err := strconv.Atoi(local_port)
	if err != nil {
		return 0, err
	}

	iremote_port, err := strconv.Atoi(remote_port)
	if err != nil {
		return 0, err
	}

	key := uint64(ilocal_port<<32) | uint64(be32(iremote_port))
	return key, nil
}

func be32(n int) uint32 {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(n))
	return binary.LittleEndian.Uint32(b[:])
}
