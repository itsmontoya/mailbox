package main

//go:generate go install github.com/joeshaw/gengen
//go:generate gengen -o mailbox_int github.com/itsmontoya/mailbox int

import (
	"fmt"

	intbox "github.com/itsmontoya/mailbox/examples/generic/mailbox_int"
)

func main() {
	const N = 5

	box := intbox.New(N)
	for i := 0; i < N; i++ {
		go box.Send(i)
	}
	for i := 0; i < N; i++ {
		n, _ := box.Receive()
		fmt.Printf("%d %T\n", n, n)
	}

	box.Close()
}
