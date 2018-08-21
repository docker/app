package com

import (
	"io"
	"io/ioutil"
	"testing"

	"context"

	protobuf "github.com/gogo/protobuf/types"
	"gotest.tools/assert"
)

type testSvc struct {
	statsToSend  []FileStat
	chunksToSend [][]byte
}

func (s *testSvc) FileContent(_ *protobuf.StringValue, stream FrontService_FileContentServer) error {
	for _, data := range s.chunksToSend {
		if err := stream.Send(&protobuf.BytesValue{Value: data}); err != nil {
			panic(err)
		}
	}
	return nil
}

func (s *testSvc) FileList(_ *protobuf.StringValue, stream FrontService_FileListServer) error {
	for _, data := range s.statsToSend {
		if err := stream.Send(&data); err != nil {
			panic(err)
		}
	}
	return nil
}

func TestComOverPipe(t *testing.T) {
	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()
	defer clientReader.Close()
	defer serverWriter.Close()
	defer serverReader.Close()
	defer clientWriter.Close()

	stdInReader, stdInWriter := io.Pipe()
	stdOutReader, stdOutWriter := io.Pipe()
	stdErrReader, stdErrWriter := io.Pipe()

	s := &testSvc{
		chunksToSend: [][]byte{
			[]byte("Hello"),
			[]byte("World"),
		},
		statsToSend: []FileStat{
			{
				Name:  "dir",
				Mode:  0755,
				IsDir: true,
			},
			{
				Name:  "file",
				Mode:  0644,
				IsDir: false,
			},
		},
	}
	go func() {
		RunFrontService(s, serverReader, serverWriter, stdInReader, stdOutWriter, stdErrWriter)
	}()
	c, remoteStreams, session, err := ConnectToFront(clientReader, clientWriter)
	assert.NilError(t, err)

	var chunksReceived [][]byte
	chunks, err := c.FileContent(context.Background(), &protobuf.StringValue{Value: ""})
	assert.NilError(t, err)
	for {
		chunk, err := chunks.Recv()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		chunksReceived = append(chunksReceived, chunk.Value)
	}
	assert.DeepEqual(t, s.chunksToSend, chunksReceived)
	var statsReceived []FileStat
	stats, err := c.FileList(context.Background(), &protobuf.StringValue{Value: ""})
	assert.NilError(t, err)
	for {
		stat, err := stats.Recv()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		statsReceived = append(statsReceived, *stat)
	}
	assert.DeepEqual(t, s.statsToSend, statsReceived)

	go func() {
		stdInWriter.Write([]byte("stdin"))
		stdInWriter.Close()
	}()

	outResult := make(chan []byte, 1)
	errResult := make(chan []byte, 1)
	defer close(outResult)
	defer close(errResult)
	go func() {
		res, err := ioutil.ReadAll(stdOutReader)
		assert.NilError(t, err)
		outResult <- res
	}()
	go func() {
		res, err := ioutil.ReadAll(stdErrReader)
		assert.NilError(t, err)
		errResult <- res
	}()

	remoteStreams.Out.Write([]byte("stdout"))
	remoteStreams.Err.Write([]byte("stderr"))

	go session.EndSession(context.Background(), &protobuf.Empty{})
	readMessage, err := ioutil.ReadAll(remoteStreams.In)
	assert.NilError(t, err)
	assert.Equal(t, "stdin", string(readMessage))
	assert.Equal(t, "stdout", string(<-outResult))
	assert.Equal(t, "stderr", string(<-errResult))
}
