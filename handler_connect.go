package socks5

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// handleConnect is used to handle a connect command
func (s *Server) handleConnect(ctx context.Context, conn conn, req *Request) error {
	// Check if this is allowed
	if ctx_, ok := s.config.Rules.Allow(ctx, req); !ok {
		if err := sendReply(conn, ruleFailure, nil); err != nil {
			return fmt.Errorf("Failed to send reply: %v", err)
		}
		return fmt.Errorf("Connect to %v blocked by rules", req.DestAddr)
	} else {
		ctx = ctx_
	}

	// Attempt to connect
	dial := s.config.Dial
	if dial == nil {
		dial = func(ctx context.Context, net_, addr string) (net.Conn, error) {
			return net.Dial(net_, addr)
		}
	}

	target, err := dial(ctx, "tcp", req.realDestAddr.Address())
	if err != nil {
		msg := err.Error()
		resp := hostUnreachable
		if strings.Contains(msg, "refused") {
			resp = connectionRefused
		} else if strings.Contains(msg, "network is unreachable") {
			resp = networkUnreachable
		}
		if err := sendReply(conn, resp, nil); err != nil {
			return fmt.Errorf("Failed to send reply: %v", err)
		}
		return fmt.Errorf("Connect to %v failed: %v", req.DestAddr, err)
	}
	defer target.Close()

	// Send success
	local := addrSpecFromNetAddr(target.LocalAddr())
	if local == nil {
		if err := sendReply(conn, serverFailure, nil); err != nil {
			return fmt.Errorf("Failed to parse AddrSpec from net.Addr: %v", target.LocalAddr())
		}
		return ErrParsingAddrSpec
	}

	if err := sendReply(conn, successReply, local); err != nil {
		return fmt.Errorf("Failed to send reply: %v", err)
	}
	return startPipe(target, conn, target, req.bufConn)
}
