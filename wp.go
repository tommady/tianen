package main

import (
	"log"
	"sync"
)

type jobFunc func() error

type workerPool struct {
	wg   *sync.WaitGroup
	pool chan jobFunc
}

func newWorkerPool(poolSize, workerNum int) *workerPool {
	wp := &workerPool{
		wg:   new(sync.WaitGroup),
		pool: make(chan jobFunc, poolSize),
	}

	for i := 0; i < workerNum; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *workerPool) worker() {
	for job := range wp.pool {
		if err := job(); err != nil {
			log.Println(err)
		}
	}
}

func (wp *workerPool) PutJob(job jobFunc) {
	wp.pool <- job
}

func (wp *workerPool) Close() {
	close(wp.pool)
	wp.wg.Wait()
}
