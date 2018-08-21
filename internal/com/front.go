package com

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	protobuf "github.com/gogo/protobuf/types"
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
func RunFrontService(impl FrontServiceServer, reader io.Reader, writer io.Writer, stdin io.ReadCloser, stdout, stderr io.WriteCloser) error {
	conn := &rawConn{
		Reader: reader,
		Writer: writer,
	}
	srv := grpc.NewServer()
	RegisterFrontServiceServer(srv, impl)
	streamServer := newRemoteStreamServer(stdin, stdout, stderr)
	RegisterRemoteStdStreamsServer(srv, streamServer)
	sessionServer := &sessionServer{
		streamServer: streamServer,
		sessionEnded: make(chan error),
	}
	RegisterSessionServer(srv, sessionServer)
	go (&http2.Server{}).ServeConn(conn, &http2.ServeConnOpts{Handler: srv})
	return <-sessionServer.sessionEnded
}

// RemoteStreams represents the standard streams of the front-end
type RemoteStreams struct {
	In      io.ReadCloser
	Out     io.WriteCloser
	Err     io.WriteCloser
	OutDone chan struct{}
	ErrDone chan struct{}
	InDone  chan struct{}
}

// ConnectToFront connects to a grpc service on a pair of reader/writer
func ConnectToFront(reader io.Reader, writer io.Writer) (FrontServiceClient, *RemoteStreams, SessionClient, error) {
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
		grpc.WithBlock(),
	}

	cc, err := grpc.Dial("", dialOpts...)
	if err != nil {
		return nil, nil, nil, err
	}

	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()
	outDone := make(chan struct{})
	inDone := make(chan struct{})
	errDone := make(chan struct{})

	streamsClient := NewRemoteStdStreamsClient(cc)
	go func() {
		defer close(inDone)
		// stdin
		defer inWriter.Close()
		input, err := streamsClient.Stdin(context.Background(), &protobuf.Empty{})
		if err != nil {
			inWriter.CloseWithError(err)
			return
		}
		for {
			message, err := input.Recv()
			switch {
			case err == io.EOF:
				return
			case err != nil:
				inWriter.CloseWithError(err)
				return
			}
			if _, err := inWriter.Write(message.Value); err != nil {
				inWriter.CloseWithError(err)
				return
			}
		}
	}()

	go func() {
		defer close(outDone)
		output, err := streamsClient.Stdout(context.Background())
		if err != nil {
			outReader.CloseWithError(err)
			return
		}
		defer output.CloseSend()

		reader := bufio.NewReader(outReader)
		buffer := make([]byte, 1024)
		for {
			read, err := reader.Read(buffer)
			switch {
			case err == io.EOF:
				return
			case err != nil:
				outReader.CloseWithError(err)
				return
			}

			if err = output.Send(&protobuf.BytesValue{Value: buffer[:read]}); err != nil {
				outReader.CloseWithError(err)
				return
			}
		}
	}()
	go func() {
		defer close(errDone)
		output, err := streamsClient.Stderr(context.Background())
		if err != nil {
			errReader.CloseWithError(err)
			return
		}
		defer output.CloseSend()

		reader := bufio.NewReader(errReader)
		buffer := make([]byte, 1024)
		for {
			read, err := reader.Read(buffer)
			switch {
			case err == io.EOF:
				return
			case err != nil:
				errReader.CloseWithError(err)
				return
			}

			if err = output.Send(&protobuf.BytesValue{Value: buffer[:read]}); err != nil {
				errReader.CloseWithError(err)
				return
			}
		}
	}()

	return NewFrontServiceClient(cc), &RemoteStreams{
		Err:     errWriter,
		Out:     outWriter,
		In:      inReader,
		ErrDone: errDone,
		OutDone: outDone,
		InDone:  inDone,
	}, NewSessionClient(cc), nil
}

// Shutdown is supposed to be called by the backend and makes sure both ends of the wire
// Have received all messages
func Shutdown(ctx context.Context, client SessionClient, streams *RemoteStreams) {
	// close streams, send end session, wait for stdin closed
	streams.Out.Close()

	streams.Err.Close()
	<-streams.OutDone
	<-streams.ErrDone

	client.EndSession(ctx, &protobuf.Empty{})
	<-streams.InDone
}

type remoteStreamServer struct {
	in      io.ReadCloser
	out     io.WriteCloser
	err     io.WriteCloser
	inDone  chan struct{}
	outDone chan struct{}
	errDone chan struct{}
}

func newRemoteStreamServer(in io.ReadCloser, out, err io.WriteCloser) *remoteStreamServer {
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()
	go func() {
		defer inWriter.Close()
		io.Copy(inWriter, in)
	}()
	go io.Copy(out, outReader)
	go io.Copy(err, errReader)
	return &remoteStreamServer{
		in:      inReader,
		out:     outWriter,
		err:     errWriter,
		inDone:  make(chan struct{}),
		outDone: make(chan struct{}),
		errDone: make(chan struct{}),
	}
}

func (s *remoteStreamServer) Stdin(_ *protobuf.Empty, output RemoteStdStreams_StdinServer) error {
	defer close(s.inDone)
	reader := bufio.NewReader(s.in)
	buffer := make([]byte, 1024)
	for {
		read, err := reader.Read(buffer)
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}

		if err = output.Send(&protobuf.BytesValue{Value: buffer[:read]}); err != nil {
			return err
		}
	}
}

type bytesReceiver interface {
	Recv() (*protobuf.BytesValue, error)
}

func bytesReceiverToOutputStream(input bytesReceiver, output io.WriteCloser) error {
	defer output.Close()
	for {
		bytes, err := input.Recv()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}
		if _, err := output.Write(bytes.Value); err != nil {
			return err
		}
	}
}

func (s *remoteStreamServer) Stdout(input RemoteStdStreams_StdoutServer) error {
	defer close(s.outDone)
	defer input.SendAndClose(&protobuf.Empty{})
	return bytesReceiverToOutputStream(input, s.out)
}
func (s *remoteStreamServer) Stderr(input RemoteStdStreams_StderrServer) error {
	defer close(s.errDone)
	defer input.SendAndClose(&protobuf.Empty{})
	return bytesReceiverToOutputStream(input, s.err)
}

type sessionServer struct {
	streamServer *remoteStreamServer
	sessionEnded chan error
}

func (s *sessionServer) EndSession(context.Context, *protobuf.Empty) (*protobuf.Empty, error) {
	defer close(s.sessionEnded)
	if err := s.streamServer.in.Close(); err != nil {
		return nil, err
	}
	<-s.streamServer.inDone
	<-s.streamServer.outDone
	<-s.streamServer.errDone

	return &protobuf.Empty{}, nil
}

func (s *sessionServer) BackendVersionMismatch(_ context.Context, vm *VersionMismatch) (*protobuf.Empty, error) {
	s.sessionEnded <- vm
	return &protobuf.Empty{}, nil
}

func (v *VersionMismatch) Error() string {
	return fmt.Sprintf("version mismatch: backend version is %q, package version is %q", v.BackendVersion, v.PackageVersion)
}
