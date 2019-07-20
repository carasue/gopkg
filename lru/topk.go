package lru

import (
	"container/heap"
	"sync"
	"time"
)

func newTopk(size int, window time.Duration, provider provider) *topk {
	p := &topk{
		ss:       (&ssalg{}).init(size),
		events:   make(chan interface{}, 1024),
		provider: provider,
	}
	go p.run(window)
	return p
}

// topk tracks top-k frequent elements in a stream.
type topk struct {
	m  sync.Map // map[interface{}]*entry
	ss *ssalg

	curkeys []interface{}
	prekeys []interface{}

	provider provider

	events chan interface{}
}

type provider func(k interface{}) (v interface{}, expires int64, ok bool)

type entry struct {
	key, value interface{}
	expires    int64
}

func (p *topk) touch(key interface{}) {
	select {
	case p.events <- key:
	default:
	}
}

func (p *topk) get(key interface{}) (v interface{}, exists, expired bool) {
	x, ok := p.m.Load(key)
	if ok {
		e := x.(*entry)
		v = e.value
		exists = true
		expired = e.expires > 0 && e.expires < time.Now().UnixNano()
	}
	return
}

func (p *topk) del(key interface{}) {
	p.m.Delete(key)
}

func (p *topk) run(window time.Duration) {
	ticker := time.NewTicker(window)
	for {
		select {
		case key := <-p.events:
			p.ss.touch(key)
		case <-ticker.C:
			ss := p.ss
			if len(p.prekeys) > 0 {
				for _, k := range p.prekeys {
					if _, ok := ss.table[k]; !ok {
						p.m.Delete(k)
					}
				}
			}
			// reuse memory
			tmp := p.prekeys
			p.prekeys = p.curkeys
			p.curkeys = tmp[:0]

			for k := range ss.table {
				p.curkeys = append(p.curkeys, k)
				v, expires, ok := p.provider(k)
				if ok {
					e := &entry{
						key:     k,
						value:   v,
						expires: expires,
					}
					p.m.Store(k, e)
				}
			}

			size := p.ss.size
			p.ss = (&ssalg{}).init(size)
		}
	}
}

// ssalg implements a modified version of the Space-Saving top-k algorithm described in:
// http://www.cse.ust.hk/~raywong/comp5331/References/EfficientComputationOfFrequentAndTop-kElementsInDataStreams.pdf
type ssalg struct {
	size     int
	table    map[interface{}]uint32
	counters []counter
	heap     ssheap
}

type counter struct {
	key      interface{}
	count    int64
	errCount int64
	idx      uint32
}

func (ss *ssalg) init(size int) *ssalg {
	*ss = ssalg{
		size:     size,
		table:    make(map[interface{}]uint32, size),
		counters: make([]counter, size),
	}
	ss.heap.h = make([]uint32, size)
	ss.heap.ss = ss
	for i := 0; i < size; i++ {
		ss.heap.h[i] = uint32(i)
		ss.counters[i].idx = uint32(i)
	}
	return ss
}

func (ss *ssalg) touch(key interface{}) {
	var counter *counter
	if idx, found := ss.table[key]; found {
		counter = &ss.counters[idx]
	} else {
		idx = uint32(ss.heap.h[0])
		counter = &ss.counters[idx]
		if counter.key != nil {
			delete(ss.table, counter.key)
		}
		counter.key = key
		counter.errCount = counter.count
		ss.table[key] = idx
	}
	counter.count++
	heap.Fix(&ss.heap, int(counter.idx))
}

type ssheap struct {
	ss *ssalg
	h  []uint32
}

func (h *ssheap) Len() int           { return len(h.h) }
func (h *ssheap) Push(x interface{}) { panic("not implemented") }
func (h *ssheap) Pop() interface{}   { panic("not implemented") }

func (h *ssheap) Less(i, j int) bool {
	ss := h.ss
	a, b := ss.counters[h.h[i]], &ss.counters[h.h[j]]
	return a.count < b.count
}

func (h *ssheap) Swap(i, j int) {
	ss := h.ss
	a, b := ss.counters[h.h[i]], ss.counters[h.h[j]]
	h.h[i], h.h[j] = h.h[j], h.h[i]
	a.idx, b.idx = uint32(j), uint32(i)
}
