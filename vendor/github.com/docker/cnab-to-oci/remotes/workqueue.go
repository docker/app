package remotes

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type workItem struct {
	do   func(ctx context.Context) error
	done chan struct{}
	deps []chan struct{}
}

func newWorkItem(do func(ctx context.Context) error, deps ...chan struct{}) *workItem {
	return &workItem{
		do:   do,
		done: make(chan struct{}),
		deps: deps,
	}
}

func (wi *workItem) process(ctx context.Context) error {
	defer close(wi.done)
	for _, d := range wi.deps {
		select {
		case <-ctx.Done():
			return nil
		case <-d:
		}
	}
	return wi.do(ctx)
}

type workQueue struct {
	workGroup *errgroup.Group
	todoList  chan *workItem
	ctx       context.Context
}

func newWorkQueue(ctx context.Context, workerCount, todoBuffer int) *workQueue {
	todoList := make(chan *workItem, todoBuffer)
	workGroup, ctx := errgroup.WithContext(ctx)
	for i := 0; i < workerCount; i++ {
		workGroup.Go(func() error {
			for {
				wi, ok := <-todoList
				if !ok {
					return nil
				}
				if err := wi.process(ctx); err != nil {
					return err
				}
			}
		})
	}
	return &workQueue{
		todoList:  todoList,
		workGroup: workGroup,
		ctx:       ctx,
	}
}

func (wq *workQueue) enqueue(do func(ctx context.Context) error, deps ...chan struct{}) chan struct{} {
	wi := newWorkItem(do, deps...)
	select {
	case <-wq.ctx.Done():
		return doneCh
	case wq.todoList <- wi:
	}
	return wi.done
}

func (wq *workQueue) stopAndWait() error {
	close(wq.todoList)
	return wq.workGroup.Wait()
}
