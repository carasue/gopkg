package lru

import (
	"log"
	"testing"
	"time"
)

func TestTopk(t *testing.T) {
	provider := func(k interface{}) (v interface{}, expires int64, ok bool) {
		return k, 0, true
	}
	ksize := 50
	window := 10 * time.Second
	topk := newTopk(ksize, window, provider)

	values := createRandInts(500)

	for i := 0; i < 3; i++ {
		go func() {
			for {
				for _, k := range values[:ksize] {
					time.Sleep(time.Millisecond)
					topk.touch(k)
				}
			}
		}()
	}
	go func() {
		for {
			for _, k := range values {
				time.Sleep(time.Millisecond)
				topk.touch(k)
			}
		}
	}()

	for {
		time.Sleep(window / 2)
		hits := make([]interface{}, 0)
		miss := make([]interface{}, 0)
		for _, k := range values[:ksize] {
			_, exists, _ := topk.get(k)
			if exists {
				hits = append(hits, k)
			} else {
				miss = append(miss, k)
			}
		}
		log.Printf("result:\n hits= %v\n miss= %v\n", hits, miss)
	}
}
