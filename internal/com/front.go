package com

import (
	"bufio"
	"context"
	"errors"
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
func RunFrontService(impl FrontServiceServer, reader io.Reader, writer io.Writer, stdin io.Reader, stdout, stderr io.WriteCloser) error {
	conn := &rawConn{
		Reader: reader,
		Writer: writer,
	}
	srv := grpc.NewServer()
	RegisterFrontServiceServer(srv, impl)
	RegisterRemoteStdStreamsServer(srv, &remoteStreamServer{
		err: stderr,
		in:  stdin,
		out: stdout,
	})
	(&http2.Server{}).ServeConn(conn, &http2.ServeConnOpts{Handler: srv})
	return nil
}

// RemoteStreams represents the standard streams of the front-end
type RemoteStreams struct {
	In  io.Reader
	Out io.WriteCloser
	Err io.WriteCloser
}

// ConnectToFront connects to a grpc service on a pair of reader/writer
func ConnectToFront(reader io.Reader, writer io.Writer) (FrontServiceClient, *RemoteStreams, error) {
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
		return nil, nil, err
	}
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()

	streamsClient := NewRemoteStdStreamsClient(cc)
	go func() {
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
		output, err := streamsClient.Stdout(context.Background())
		defer output.CloseSend()
		if err != nil {
			outReader.CloseWithError(err)
			return
		}

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
		output, err := streamsClient.Stderr(context.Background())
		defer output.CloseSend()
		if err != nil {
			errReader.CloseWithError(err)
			return
		}

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
		Err: errWriter,
		Out: outWriter,
		In:  inReader,
	}, nil
}

type remoteStreamServer struct {
	in  io.Reader
	out io.WriteCloser
	err io.WriteCloser
}

func (s *remoteStreamServer) Stdin(_ *protobuf.Empty, output RemoteStdStreams_StdinServer) error {
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
	return bytesReceiverToOutputStream(input, s.out)
}
func (s *remoteStreamServer) Stderr(input RemoteStdStreams_StderrServer) error {
	return bytesReceiverToOutputStream(input, s.err)
}
