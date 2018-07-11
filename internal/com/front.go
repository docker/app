package com

import (
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
	"google.golang.org/grpc"
)

type rawConn struct {
	io.Reader
	io.Writer
}

func (c *rawConn) Close() error {
	return nil // noop, reader/writer lifetyme handled by caller
}

func (c *rawConn) LocalAddr() net.Addr {
	return &net.IPAddr{IP: net.IPv4zero}
}

func (c *rawConn) RemoteAddr() net.Addr {
	return &net.IPAddr{IP: net.IPv4zero}
}
func (c *rawConn) SetDeadline(t time.Time) error {
	return errors.New("deadline not implemented")
}
func (c *rawConn) SetReadDeadline(t time.Time) error {
	return errors.New("deadline not implemented")
}
func (c *rawConn) SetWriteDeadline(t time.Time) error {
	return errors.New("deadline not implemented")
}

var _ net.Conn = &rawConn{}

// RunFrontService host a grpc service on a pair of reader/writer
func RunFrontService(impl FrontServiceServer, reader io.Reader, writer io.Writer) error {
	conn := &rawConn{
		Reader: reader,
		Writer: writer,
	}
	srv := grpc.NewServer()
	RegisterFrontServiceServer(srv, impl)
	(&http2.Server{}).ServeConn(conn, &http2.ServeConnOpts{Handler: srv})
	return nil
}

// ConnectToFront connects to a grpc service on a pair of reader/writer
func ConnectToFront(reader io.Reader, writer io.Writer) (FrontServiceClient, error) {
	conn := &rawConn{
		Reader: reader,
		Writer: writer,
	}
	var dialCount int64
	dialer := grpc.WithDialer(func(addr string, d time.Duration) (net.Conn, error) {
		if c := atomic.AddInt64(&dialCount, 1); c > 1 {
			return nil, errors.New("only one connection allowed")
		}
		return conn, nil
	})

	dialOpts := []grpc.DialOption{
		dialer,
		grpc.WithInsecure(),
	}

	cc, err := grpc.Dial("", dialOpts...)
	if err != nil {
		return nil, err
	}
	return NewFrontServiceClient(cc), nil
}
