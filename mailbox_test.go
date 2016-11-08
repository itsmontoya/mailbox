package mailbox

import (
	"sync"
	"testing"

	"github.com/joeshaw/gengen/generic"
)

var (
	testVal int

	testBufSize = 128
	testSet     = getList(8192)
	testBatch   = getList(512)
)

func TestMailbox(t *testing.T) {
	var wg sync.WaitGroup
	var cnt int
	wg.Add(2)
	mb := New(testBufSize)

	go func() {
		mb.Listen(func(item generic.T) (end bool) {
			var (
				v  int
				ok bool
			)

			if v, ok = item.(int); !ok {
				panic("Whoa")
			}

			testVal = v
			cnt++
			return
		})

		wg.Done()
	}()

	go func() {
		for _, si := range testSet {
			mb.Send(si)
		}
		mb.Close()
		wg.Done()
	}()

	wg.Wait()
	if cnt != len(testSet) {
		t.Fatal("Errr cnt", cnt)
	}
}

func BenchmarkMailbox(b *testing.B) {
	var rwg sync.WaitGroup
	mb := New(testBufSize)
	rwg.Add(1)

	go func() {
		var (
			v  int
			ok bool
		)

		mb.Listen(func(item generic.T) (end bool) {

			if v, ok = item.(int); !ok {
				panic("Whoa")
			}

			testVal = v
			return
		})
		rwg.Done()
	}()

	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			mb.Send(i)
			i++
		}
	})

	mb.Close()
	rwg.Wait()

	b.ReportAllocs()
}

func BenchmarkChannel(b *testing.B) {
	var rwg sync.WaitGroup
	ch := make(chan generic.T, testBufSize)
	rwg.Add(1)

	go func() {
		var (
			v  int
			ok bool
		)

		for item := range ch {
			if v, ok = item.(int); !ok {
				panic("Whoa")
			}

			testVal = v
		}

		rwg.Done()
	}()

	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			ch <- i
			i++
		}
	})

	close(ch)
	rwg.Wait()

	b.ReportAllocs()
}

func BenchmarkBatchMailbox(b *testing.B) {
	var rwg sync.WaitGroup
	mb := New(testBufSize)
	rwg.Add(1)

	go func() {
		mb.Listen(func(item generic.T) (end bool) {
			var (
				v  int
				ok bool
			)

			if v, ok = item.(int); !ok {
				panic("Whoa")
			}

			testVal = v
			return
		})
		rwg.Done()
	}()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mb.Batch(testBatch...)
		}
	})

	mb.Close()
	rwg.Wait()

	b.ReportAllocs()
}

func BenchmarkBatchChannel(b *testing.B) {
	var rwg sync.WaitGroup
	ch := make(chan generic.T, testBufSize)
	rwg.Add(1)

	go func() {
		for item := range ch {
			var (
				v  int
				ok bool
			)

			if v, ok = item.(int); !ok {
				panic("Whoa")
			}

			testVal = v
		}

		rwg.Done()
	}()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, i := range testBatch {
				ch <- i
			}
		}
	})

	close(ch)
	rwg.Wait()

	b.ReportAllocs()
}

func getList(n int) (l []generic.T) {
	l = make([]generic.T, n)
	for i := range l {
		l[i] = i
	}

	return
}
