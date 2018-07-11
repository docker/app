package com

import (
	"io"
	"testing"

	"context"

	protobuf "github.com/gogo/protobuf/types"
	"gotest.tools/assert"
)

type testSvc struct {
	promptedText string
	promptInput  string
	printedText  string
	statsToSend  []FileStat
	chunksToSend [][]byte
}

func (s *testSvc) Prompt(ctx context.Context, in *protobuf.BytesValue) (*protobuf.BytesValue, error) {
	s.promptedText = string(in.Value)
	return &protobuf.BytesValue{Value: []byte(s.promptInput)}, nil
}

func (s *testSvc) Print(ctx context.Context, in *PrintRequest) (*protobuf.Empty, error) {
	s.printedText = string(in.Bytes)
	return &protobuf.Empty{}, nil
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
	s := &testSvc{promptInput: "response",
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
		RunFrontService(s, serverReader, serverWriter)
	}()
	c, err := ConnectToFront(clientReader, clientWriter)
	assert.NilError(t, err)
	res, err := c.Prompt(context.Background(), &protobuf.BytesValue{Value: []byte("request")})
	assert.NilError(t, err)
	assert.Equal(t, string(res.Value), "response")
	assert.Equal(t, s.promptedText, "request")
	_, err = c.Print(context.Background(), &PrintRequest{Bytes: []byte("hello")})
	assert.NilError(t, err)
	assert.Equal(t, s.printedText, "hello")

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
}
