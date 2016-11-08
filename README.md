# mailbox [![GoDoc](https://godoc.org/github.com/itsmontoya/mailbox?status.svg)](https://godoc.org/github.com/itsmontoya/mailbox) ![Status](https://img.shields.io/badge/status-beta-yellow.svg)

mailbox a send and receive library, it's simple with few frills. 

## Benchmarks
```
BenchmarkMailbox-4              20000000            85.7 ns/op         8 B/op      1 allocs/op
BenchmarkChannel-4              10000000           175 ns/op           8 B/op      1 allocs/op
BenchmarkBatchMailbox-4           100000         14272 ns/op           0 B/op      0 allocs/op
BenchmarkBatchChannel-4            20000         66022 ns/op           0 B/op      0 allocs/op
```

## Usage
``` go
package main

import (
        "fmt"

        "github.com/itsmontoya/mailbox"
)

func main() {
        mb := mailbox.New(32)

        go func() {
                mb.Send("Hello world!")
                mb.Close()
        }()

        mb.Listen(func(msg interface{}) (end bool) {
                fmt.Println(msg)
                return
        })
}
```