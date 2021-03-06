package beanstalk

import "sync"

// ProducerPool maintains a pool of Producers with the purpose of spreading
// incoming Put requests over the maintained Producers.
type ProducerPool struct {
	producers []*Producer
	putC      chan *Put
	putTokens chan *Put
	stopOnce  sync.Once
}

// NewProducerPool creates a pool of Producer objects.
func NewProducerPool(urls []string, options *Options) (*ProducerPool, error) {
	pool := &ProducerPool{putC: make(chan *Put)}
	pool.putTokens = make(chan *Put, len(urls))

	for _, url := range urls {
		producer, err := NewProducer(url, pool.putC, options)
		if err != nil {
			return nil, err
		}

		pool.producers = append(pool.producers, producer)
		pool.putTokens <- NewPut(pool.putC, options)
	}

	for _, producer := range pool.producers {
		producer.Start()
	}

	return pool, nil
}

// Stop shuts down all the producers in the pool.
func (pool *ProducerPool) Stop() {
	pool.stopOnce.Do(func() {
		for i, producer := range pool.producers {
			producer.Stop()
			pool.producers[i] = nil
		}

		pool.producers = []*Producer{}
	})
}

// Put inserts a new job into beanstalk.
func (pool *ProducerPool) Put(tube string, body []byte, params *PutParams) (uint64, error) {
	put := <-pool.putTokens
	id, err := put.Request(tube, body, params)
	pool.putTokens <- put

	return id, err
}
