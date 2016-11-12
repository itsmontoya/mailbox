package mailbox

import (
	"sync"
	"testing"
)

var (
	testVal *byte

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
		mb.Listen(func(item *byte) (end bool) {
			testVal = item
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
		mb.Listen(func(item *byte) (end bool) {
			testVal = item
			return
		})
		rwg.Done()
	}()

	b.RunParallel(func(pb *testing.PB) {
		var i *byte
		for pb.Next() {
			mb.Send(i)
		}
	})

	mb.Close()
	rwg.Wait()

	b.ReportAllocs()
}

func BenchmarkChannel(b *testing.B) {
	var rwg sync.WaitGroup
	ch := make(chan *byte, testBufSize)
	rwg.Add(1)

	go func() {
		for item := range ch {
			testVal = item
		}

		rwg.Done()
	}()

	b.RunParallel(func(pb *testing.PB) {
		var i *byte
		for pb.Next() {
			ch <- i
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
		mb.Listen(func(item *byte) (end bool) {
			testVal = item
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
	ch := make(chan *byte, testBufSize)
	rwg.Add(1)

	go func() {
		for item := range ch {
			testVal = item
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

func getList(n int) (l []*byte) {
	l = make([]*byte, n)
	for i := range l {
		l[i] = empty
	}

	return
}
