# mailbox [![GoDoc](https://godoc.org/github.com/itsmontoya/mailbox?status.svg)](https://godoc.org/github.com/mailbox/oBandit) ![Status](https://img.shields.io/badge/status-beta-yellow.svg)

mailbox a send and receive library, it's simple with few frills. 

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