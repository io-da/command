package command

import (
	"errors"
	"fmt"
)

// Async is the struct returned from async commands.
type AsyncList struct {
	cmds []*Async
}

func NewAsyncList(asyncList ...*Async) *AsyncList {
	return &AsyncList{
		cmds: asyncList,
	}
}

// Push appends the provided async commands to the async list
func (asl *AsyncList) Push(asyncList ...*Async) {
	asl.cmds = append(asl.cmds, asyncList...)
}

// Await waits for the async commands to be processed and returns their results/errors respectively.
func (asl *AsyncList) Await() ([]any, error) {
	iterator, err := asl.AwaitIterator()
	if err != nil {
		return nil, err
	}

	data := make([]any, len(asl.cmds))
	errs := make([]error, len(asl.cmds))
	for res := range iterator {
		data[res.Index], errs[res.Index] = res.Get()
	}

	return data, errors.Join(errs...)
}

// AwaitIterator generates an iterator to iterate over the await results in order of arrival.
func (asl *AsyncList) AwaitIterator() (<-chan AsyncResult, error) {
	if len(asl.cmds) == 0 {
		return nil, EmptyAwaitListError
	}
	results := make(chan AsyncResult, len(asl.cmds))
	processed := newCounter()
	total := uint32(len(asl.cmds))
	for i := 0; i < len(asl.cmds); i++ {
		asl.cmds[i].setListener(asl.generateListener(i, results, processed, total))
	}
	return results, nil
}

func (asl *AsyncList) generateListener(i int, results chan<- AsyncResult, processed *counter, total uint32) func(as *Async) {
	return func(as *Async) {
		results <- AsyncResult{
			Index: i,
			Data:  as.data,
			Err:   as.err,
		}
		if processed.increment() == total {
			close(results)
			fmt.Println("CHANNEL CLOSED")
		}
	}
}
