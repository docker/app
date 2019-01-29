// +build go1.12

package fifo

import (
	"syscall"

	"github.com/pkg/errors"
)

// SyscallConn provides raw access to the fifo's underlying filedescrptor.
// See syscall.Conn for guarentees provided by this interface.
func (f *fifo) SyscallConn() (syscall.RawConn, error) {
	// deterministic check for closed
	select {
	case <-f.closed:
		return nil, errors.New("fifo closed")
	default:
	}

	select {
	case <-f.closed:
		return nil, errors.New("fifo closed")
	case <-f.opened:
		return f.file.SyscallConn()
	default:
	}

	// Not opened and not closed, this means open is non-blocking AND it's not open yet
	// Use rawConn to deal with non-blocking open.
	rc := &rawConn{f: f, ready: make(chan struct{})}
	go func() {
		select {
		case <-f.closed:
			return
		case <-f.opened:
			rc.raw, rc.err = f.file.SyscallConn()
			close(rc.ready)
		}
	}()

	return rc, nil
}

type rawConn struct {
	f     *fifo
	ready chan struct{}
	raw   syscall.RawConn
	err   error
}

func (r *rawConn) Control(f func(fd uintptr)) error {
	select {
	case <-r.f.closed:
		return errors.New("control of closed fifo")
	case <-r.ready:
	}

	if r.err != nil {
		return r.err
	}

	return r.raw.Control(f)
}

func (r *rawConn) Read(f func(fd uintptr) (done bool)) error {
	if r.f.flag&syscall.O_WRONLY > 0 {
		return errors.New("reading from write-only fifo")
	}

	select {
	case <-r.f.closed:
		return errors.New("reading of a closed fifo")
	case <-r.ready:
	}

	if r.err != nil {
		return r.err
	}

	return r.raw.Read(f)
}

func (r *rawConn) Write(f func(fd uintptr) (done bool)) error {
	if r.f.flag&(syscall.O_WRONLY|syscall.O_RDWR) == 0 {
		return errors.New("writing to read-only fifo")
	}

	select {
	case <-r.f.closed:
		return errors.New("writing to a closed fifo")
	case <-r.ready:
	}

	if r.err != nil {
		return r.err
	}

	return r.raw.Write(f)
}
