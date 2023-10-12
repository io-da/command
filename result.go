package command

import (
	"sync"
)

// Result is the struct returned from regular queries.
type Result struct {
	sync.Mutex
	data any
}

func newResult() *Result {
	return &Result{}
}

//------Fetch Data------//

// Get retrieves the data from the return of the first command.
func (res *Result) Get() any {
	return res.data
}

//------Internal------//

func (res *Result) setReturn(data any) {
	res.data = data
}
