package remotes

import (
	"context"
	"reflect"

	"golang.org/x/sync/errgroup"
)

// scheduler is an abstraction over a component capable of running tasks
// concurrently
type scheduler interface {
	// schedule a task and returns a promise notifying completion
	schedule(func(ctx context.Context) error) promise
	// ctx returns the context associated with this scheduler
	ctx() context.Context
}

// dependency represents an asynchronous operation with its completion channel and its error state
type dependency interface {
	Done() <-chan struct{}
	Err() error
}

// failedDependency is a dependency already ran to completion with an error
type failedDependency struct {
	err error
}

func (f failedDependency) Done() <-chan struct{} {
	return doneCh
}

func (f failedDependency) Err() error {
	return f.err
}

// doneDependency is a dependency already ran to completion without error
type doneDependency struct {
}

func (doneDependency) Done() <-chan struct{} {
	return doneCh
}

func (doneDependency) Err() error {
	return nil
}

// doneCh is used internally by doneDependency and failedDependency (it is a closed channel)
var doneCh = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// whenAll wrapps multiple dependencies in a single dependency
// the result is completed once any dependency completes with an error
// or once all dependencies ran to completion without error
func whenAll(dependencies []dependency) dependency {
	completionSource := &completionSource{
		done: make(chan struct{}),
	}
	go func() {
		defer close(completionSource.done)
		cases := make([]reflect.SelectCase, len(dependencies))
		for ix, dependency := range dependencies {
			cases[ix] = reflect.SelectCase{
				Chan: reflect.ValueOf(dependency.Done()),
				Dir:  reflect.SelectRecv,
			}
		}
		for len(dependencies) > 0 {
			ix, _, _ := reflect.Select(cases)
			if err := dependencies[ix].Err(); err != nil {
				completionSource.err = err
				return
			}
			cases = append(cases[:ix], cases[ix+1:]...)
			dependencies = append(dependencies[:ix], dependencies[ix+1:]...)
		}
	}()
	return completionSource
}

// promise is a dependency attached to a scheduler. It allows to schedule continuations
type promise struct {
	dependency
	scheduler scheduler
}

func (p promise) wait() error {
	<-p.Done()
	return p.Err()
}

// then schedules a continuation task once the current promise is completed.
// It propagates errors and returns a promise wrapping the continuation
func (p promise) then(next func(ctx context.Context) error) promise {
	completionSource := &completionSource{
		done: make(chan struct{}),
	}
	go func() {
		defer close(completionSource.done)
		<-p.Done()
		if err := p.Err(); err != nil {
			completionSource.err = err
			return
		}
		completionSource.err = p.scheduler.schedule(next).wait()
	}()
	return newPromise(p.scheduler, completionSource)
}

// newPromise creates a promise out of a dependency
func newPromise(scheduler scheduler, dependency dependency) promise {
	return promise{scheduler: scheduler, dependency: dependency}
}

// this schedule a task that itself produces a promise, and returns a promise wrapping the produced promise
func scheduleAndUnwrap(scheduler scheduler, do func(ctx context.Context) (dependency, error)) promise {
	completionSource := &completionSource{
		done: make(chan struct{}),
	}
	scheduler.schedule(func(ctx context.Context) error {
		p, err := do(ctx)
		if err != nil {
			completionSource.err = err
			close(completionSource.done)
			return err
		}
		go func() {
			<-p.Done()
			completionSource.err = p.Err()
			close(completionSource.done)
		}()
		return nil
	})
	return newPromise(scheduler, completionSource)
}

// completion source is a a low-level dependency implementation used internally by the schedulers and promises
type completionSource struct {
	done chan struct{}
	err  error
}

func (cs *completionSource) Done() <-chan struct{} {
	return cs.done
}

func (cs *completionSource) Err() error {
	return cs.err
}

// todoItem is an internal structure used by errgroupScheduler
type todoItem struct {
	completionSource *completionSource
	do               func(ctx context.Context) error
}

// errgroupScheduler is a scheduler that cancels all tasks at the first error occurred
type errgroupScheduler struct {
	workGroup *errgroup.Group
	todoList  chan todoItem
	context   context.Context
}

func newErrgroupScheduler(ctx context.Context, workerCount, todoBuffer int) *errgroupScheduler {
	todoList := make(chan todoItem, todoBuffer)
	workGroup, ctx := errgroup.WithContext(ctx)
	for i := 0; i < workerCount; i++ {
		workGroup.Go(func() error {
			for {
				select {
				case todoItem := <-todoList:
					todoItem.completionSource.err = todoItem.do(ctx)
					close(todoItem.completionSource.done)
					if todoItem.completionSource.err != nil {
						return todoItem.completionSource.err
					}
				case <-ctx.Done():
					return ctx.Err()
				}

			}
		})
	}
	return &errgroupScheduler{
		todoList:  todoList,
		workGroup: workGroup,
		context:   ctx,
	}
}

func (s *errgroupScheduler) schedule(do func(ctx context.Context) error) promise {
	select {
	case <-s.context.Done():
		return newPromise(s, failedDependency{s.context.Err()})
	default:
	}
	completionSource := &completionSource{
		done: make(chan struct{}),
	}
	s.todoList <- todoItem{completionSource: completionSource, do: do}
	return newPromise(s, completionSource)
}

func (s *errgroupScheduler) ctx() context.Context {
	return s.context
}

// nolint: unparam
func (s *errgroupScheduler) drain() error {
	return s.workGroup.Wait()
}
