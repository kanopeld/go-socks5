package socks5

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"os"
)

const (
	socks5Version = uint8(5)
)

// Config is used to setup and configure a Server
type Config struct {
	// AuthMethods can be provided to implement custom authentication
	// By default, "auth-less" mode is enabled.
	// For password-based auth use UserPassAuthenticator.
	AuthMethods []Authenticator

	// If provided, username/password authentication is enabled,
	// by appending a UserPassAuthenticator to AuthMethods. If not provided,
	// and AUthMethods is nil, then "auth-less" mode is enabled.
	Credentials CredentialStore

	// Resolver can be provided to do custom name resolution.
	// Defaults to DNSResolver if not provided.
	Resolver NameResolver

	// Rules is provided to enable custom logic around permitting
	// various commands. If not provided, PermitAll is used.
	Rules RuleSet

	// Rewriter can be used to transparently rewrite addresses.
	// This is invoked before the RuleSet is invoked.
	// Defaults to NoRewrite.
	Rewriter AddressRewriter

	// BindIP is used for bind or udp associate
	BindIP net.IP

	// Logger can be used to provide a custom log target.
	// Defaults to stdout.
	Logger *logrus.Logger

	// Optional function for dialing out
	Dial DialFunc

	Listener ListenerFunc
}

type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

type ListenerFunc func(ctx context.Context, network, port string) (net.Listener, error)

// Server is reponsible for accepting connections and handling
// the details of the SOCKS5 protocol
type Server struct {
	config      *Config
	authMethods map[uint8]Authenticator
}

// New creates a new Server and potentially returns an error
func New(conf *Config) (*Server, error) {
	// Ensure we have at least one authentication method enabled
	if len(conf.AuthMethods) == 0 {
		if conf.Credentials != nil {
			conf.AuthMethods = []Authenticator{&UserPassAuthenticator{conf.Credentials}}
		} else {
			conf.AuthMethods = []Authenticator{&NoAuthAuthenticator{}}
		}
	}

	// Ensure we have a DNS resolver
	if conf.Resolver == nil {
		conf.Resolver = DNSResolver{}
	}

	// Ensure we have a rule set
	if conf.Rules == nil {
		conf.Rules = PermitAll()
	}

	// Ensure we have a log target
	if conf.Logger == nil {
		conf.Logger = logrus.New()
		conf.Logger.Out = os.Stdout
	}

	server := &Server{
		config: conf,
	}

	server.authMethods = make(map[uint8]Authenticator)

	for _, a := range conf.AuthMethods {
		server.authMethods[a.GetCode()] = a
	}

	return server, nil
}

// ListenAndServe is used to create a listener and serve on it
func (s *Server) ListenAndServe(network, addr string) error {
	l, err := net.Listen(network, addr)
	if err != nil {
		s.log().WithField("err", err.Error()).Error("start listen")
		return err
	}
	s.log().WithField("addr", addr).Info("listener started")
	return s.Serve(l)
}

// Serve is used to serve connections from a listener
func (s *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			s.log().WithField("err", err.Error()).Debug("new connection")
			return err
		}
		s.log().WithFields(logrus.Fields{
			"addrR": conn.RemoteAddr().String(),
			"addrL": conn.LocalAddr().String(),
		}).Info("new connection")
		go s.ServeConn(conn)
	}
}

// ServeConn is used to serve a single connection.
func (s *Server) ServeConn(conn net.Conn) error {
	defer conn.Close()
	bufConn := bufio.NewReader(conn)

	// Read the version byte
	version := []byte{0}
	if _, err := bufConn.Read(version); err != nil {
		s.log().WithFields(logrus.Fields{
			"addrR": conn.RemoteAddr().String(),
			"addrL": conn.LocalAddr().String(),
			"err":   err.Error(),
		}).Error("failed to get version byte")
		return err
	}

	// Ensure we are compatible
	if version[0] != socks5Version {
		err := fmt.Errorf("unsupported SOCKS version: %v", version)
		s.log().WithFields(logrus.Fields{
			"addrR": conn.RemoteAddr().String(),
			"addrL": conn.LocalAddr().String(),
			"err":   err,
		}).Error("socks")
		return err
	}

	// Authenticate the connection
	authContext, err := s.authenticate(conn, bufConn)
	if err != nil {
		err = fmt.Errorf("failed to authenticate: %v", err)
		s.log().WithFields(logrus.Fields{
			"addrR": conn.RemoteAddr().String(),
			"addrL": conn.LocalAddr().String(),
			"err":   err,
		}).Error("socks")
		return err
	}

	request, err := NewRequest(bufConn)
	if err != nil {
		if err == unrecognizedAddrType {
			if err := sendReply(conn, addrTypeNotSupported, nil); err != nil {
				err = fmt.Errorf("failed to send reply: %v", err)
				s.log().WithFields(logrus.Fields{
					"addrR": conn.RemoteAddr().String(),
					"addrL": conn.LocalAddr().String(),
					"err":   err,
				}).Error("socks")
				return err
			}
		}
		err = fmt.Errorf("failed to read destination address: %v", err)
		s.log().WithFields(logrus.Fields{
			"addrR": conn.RemoteAddr().String(),
			"addrL": conn.LocalAddr().String(),
			"err":   err,
		}).Error("socks")
		return err
	}
	request.AuthContext = authContext
	if client, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		request.RemoteAddr = &AddrSpec{IP: client.IP, Port: client.Port}
	}

	// Process the client request
	if err := s.handleRequest(request, conn); err != nil {
		s.log().WithFields(logrus.Fields{
			"addrR": conn.RemoteAddr().String(),
			"addrL": conn.LocalAddr().String(),
		}).Info("waiting for jumpbox to be available")
		return err
	}
	return nil
}

func (s *Server) log() *logrus.Logger {
	return s.config.Logger
}
