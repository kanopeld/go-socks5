package socks5

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// handleBind is used to handle a connect command
func (s *Server) handleBind(ctx context.Context, conn conn, req *Request) error {
	// Check if this is allowed
	if ctx_, ok := s.config.Rules.Allow(ctx, req); !ok {
		if err := sendReply(conn, ruleFailure, nil); err != nil {
			return fmt.Errorf("Failed to send reply: %v", err)
		}
		return fmt.Errorf("Bind to %v blocked by rules", req.DestAddr)
	} else {
		ctx = ctx_
	}

	listener := s.config.Listener
	if listener == nil {
		listener = func(ctx context.Context, network, port string) (net.Listener, error) {
			return net.Listen(network, port)
		}
	}

	ln, err := listener(ctx, "tcp", fmt.Sprintf(":%d", req.realDestAddr.Port))
	if err != nil || ln == nil {
		if err := sendReply(conn, serverFailure, nil); err != nil {
			return fmt.Errorf("Failed to parse AddrSpec from net.Addr: %v", ln.Addr().String())
		}
	}
	defer ln.Close()

	local := addrSpecFromNetAddr(ln.Addr())
	if local == nil {
		if err := sendReply(conn, serverFailure, nil); err != nil {
			return fmt.Errorf("Failed to parse AddrSpec from net.Addr: %v", ln.Addr())
		}
		return ErrParsingAddrSpec
	}

	if err := sendReply(conn, successReply, local); err != nil {
		return fmt.Errorf("Failed to send reply: %v", err)
	}

	go func() {
		mux := new(sync.Mutex)
		local := conn.(net.Conn)

		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				mux.Lock()
				defer mux.Unlock()

				_ = startPipe(local, conn, local, conn)
			}()
		}
	}()

	return nil
}
