package socks5

import "net"

type bindListener struct {
	ln    net.Listener
	local net.Conn
}

func newBindListener(ln net.Listener, conn net.Conn) *bindListener {
	return &bindListener{
		ln:    ln,
		local: conn,
	}
}

func (bl *bindListener) start() {
	for {
		conn, err := bl.ln.Accept()
		if err != nil {
			return
		}

	}
}
