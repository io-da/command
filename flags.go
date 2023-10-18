package command

import "sync/atomic"

type flag struct {
	atomic.Uint32
}

func newFlag() *flag {
	return &flag{
		Uint32: atomic.Uint32{},
	}
}

func (flg *flag) enabled() bool {
	return flg.Load() == 1
}

func (flg *flag) enable() (swapped bool) {
	return flg.CompareAndSwap(0, 1)
}

func (flg *flag) disable() (swapped bool) {
	return flg.CompareAndSwap(1, 0)
}
