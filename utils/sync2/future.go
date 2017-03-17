package sync2

import "sync"

type Future struct {
	sync.Mutex
	wait sync.WaitGroup
	vmap map[string]interface{}
}

/*
	初始化
	return: map[string]interface{}
 */
func (f *Future) lazyInit() {
	if f.vmap == nil {
		f.vmap = make(map[string]interface{})
	}
}

/*
	Add
	sync.WaitGroup + 1
 */
func (f *Future) Add() {
	f.wait.Add(1)
}

/*
	map[string]interface{}
	赋值：vmap[key] = val

 */
func (f *Future) Done(key string, val interface{}) {
	f.Lock()
	defer f.Unlock()
	f.lazyInit()
	f.vmap[key] = val
	f.wait.Done()
}

/*
	return map[string]interface{}
 */
func (f *Future) Wait() map[string]interface{} {
	f.wait.Wait()
	f.Lock()
	defer f.Unlock()
	f.lazyInit()
	return f.vmap
}
