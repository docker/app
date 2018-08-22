package fs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/docker/docker/pkg/archive"

	"github.com/docker/app/internal/com"
	protobuf "github.com/gogo/protobuf/types"
)

// FrontFileServer is an implementation of the the remote FS exposed by the frontend
type FrontFileServer struct {
}

// FileContent retrieve the content of a single file
func (FrontFileServer) FileContent(path *protobuf.StringValue, chunkSink com.FrontService_FileContentServer) error {
	f, err := os.Open(path.Value)
	if err != nil {
		return err
	}
	defer f.Close()
	buffer := make([]byte, 4096)
	for {
		read, err := f.Read(buffer)
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}
		fmt.Printf("received %d bytes\n", read)
		if err = chunkSink.Send(&protobuf.BytesValue{Value: buffer[:read]}); err != nil {
			return err
		}
	}
}

// FileList list the files of a directory
func (FrontFileServer) FileList(path *protobuf.StringValue, statSink com.FrontService_FileListServer) error {
	stats, err := ioutil.ReadDir(path.Value)
	if err != nil {
		return err
	}
	for _, stat := range stats {
		if err = statSink.Send(&com.FileStat{
			IsDir: stat.IsDir(),
			Mode:  int32(stat.Mode()),
			Name:  stat.Name(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// TarDir tar the content of a directory
func (FrontFileServer) TarDir(path *protobuf.StringValue, chunkSink com.FrontService_TarDirServer) error {
	arch, err := archive.Tar(path.Value, archive.Uncompressed)
	if err != nil {
		return err
	}
	defer arch.Close()
	buffer := make([]byte, 4096)
	for {
		read, err := arch.Read(buffer)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err = chunkSink.Send(&protobuf.BytesValue{Value: buffer[:read]}); err != nil {
			return err
		}
	}
}

// UntarDir untar the given data stream
func (FrontFileServer) UntarDir(chunkSource com.FrontService_UntarDirServer) error {
	firstChunk, err := chunkSource.Recv()
	if err != nil {
		return err
	}
	reader, writer := io.Pipe()
	defer writer.Close()
	done := make(chan error)
	go func() {
		defer close(done)
		done <- archive.Untar(reader, firstChunk.Dest, nil)
	}()
	if _, err := writer.Write(firstChunk.Data); err != nil {
		return err
	}
	for {
		data, err := chunkSource.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if _, err := writer.Write(data.Data); err != nil {
			return err
		}
	}
	return <-done
}
