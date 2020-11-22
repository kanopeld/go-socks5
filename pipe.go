package socks5

import "io"

type closeWriter interface {
	CloseWrite() error
}

// proxy is used to suffle data from src to destination, and sends errors
// down a dedicated channel
func proxy(dst io.Writer, src io.Reader, errCh chan error) {
	_, err := io.Copy(dst, src)
	if tcpConn, ok := dst.(closeWriter); ok {
		_ = tcpConn.CloseWrite()
	}
	errCh <- err
}

// startPipe starts the communication process between local and remote connections
func startPipe(targetWriter, localWriter io.Writer, targetReader, localReader io.Reader) error {
	// Start proxying
	errCh := make(chan error, 2)
	go proxy(targetWriter, localReader, errCh)
	go proxy(localWriter, targetReader, errCh)

	// Wait
	for i := 0; i < 2; i++ {
		e := <-errCh
		if e != nil {
			// return from this function closes target (and conn).
			return e
		}
	}
	return nil
}
