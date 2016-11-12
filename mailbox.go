package mailbox

import (
	"sync"
	"sync/atomic"

	"github.com/joeshaw/gengen/generic"
)

// New returns a new instance of Mailbox
func New(sz int) *Mailbox {
	mb := Mailbox{
		cap:  sz,
		tail: -1,

		s: make([]generic.T, sz),
	}

	// Initialize the conds
	mb.sc = sync.NewCond(&mb.mux)
	mb.rc = sync.NewCond(&mb.mux)
	return &mb
}

// Mailbox is used to send and receive messages
type Mailbox struct {
	mux sync.Mutex
	sc  *sync.Cond
	rc  *sync.Cond

	s []generic.T

	len  int
	cap  int
	head int
	tail int

	closed int32
}

func (m *Mailbox) isClosed() bool {
	return atomic.LoadInt32(&m.closed) == 1
}

// rWait is a wait function for receivers
func (m *Mailbox) rWait(wait bool) (state StateCode) {
	for m.len == 0 {
		if m.isClosed() {
			// Our inbox is empty AND closed, return StateClosed
			return StateClosed
		}

		if !wait {
			// We aren't waiting for new messages, return StateEmpty
			return StateEmpty
		}

		// Let's wait for a signal..
		m.rc.Wait()
	}

	// We waited for an available message, return StateOK
	return
}

var empty generic.T

// receive is the internal function for receiving messages
func (m *Mailbox) receive(wait bool) (msg generic.T, state StateCode) {
	if state = m.rWait(wait); state != StateOK {
		return
	}

	// Set message as the current head
	msg = m.s[m.head]
	// Empty the current head value to avoid any retainment issues
	m.s[m.head] = empty
	// Goto the next index
	if m.head++; m.head == m.cap {
		// Our increment falls out of the bounds of our internal slice, reset to 0
		m.head = 0
	}

	// Decrement the length
	if m.len--; m.len == m.cap-1 {
		// Notify the senders that we have a vacant entry
		m.sc.Broadcast()
	}

	return
}

func (m *Mailbox) sWait(wait bool) (state StateCode) {
	for m.cap-m.len == 0 {
		if !wait {
			return StateFull
		}

		// There are no vacant spots in the inbox, time to wait
		m.sc.Wait()
	}

	// An entry is available, return StateOK
	return
}

// send is the internal function used for sending messages, if the list is full:
//  - If wait is true, will wait for an available space
//  - Else, will return will early with a state of StateFull
func (m *Mailbox) send(msg generic.T, wait bool) (state StateCode) {
	if state = m.sWait(wait); state != StateOK {
		return
	}

	// Increment tail index
	m.incTail()
	// Send the new tail as the provided message
	m.s[m.tail] = msg
	// Increment the length
	m.incLen()
	return
}

// pop will append a new message to the end of the list
// If the list is full, the oldest message will be overwritten
func (m *Mailbox) pop(msg generic.T) {
	// Increment tail index
	m.incTail()
	// Send the new tail as the provided message
	m.s[m.tail] = msg
	// Increment the length
	m.incLen()
}

func (m *Mailbox) incTail() {
	// Goto the next index
	if m.tail++; m.tail == m.cap {
		// Our increment falls out of the bounds of our internal slice, reset to 0
		m.tail = 0
	}
}

func (m *Mailbox) incLen() {
	// Increment the length
	if m.len++; m.len == 1 {
		// Notify the receivers that we new message
		m.rc.Broadcast()
	}
}

// Send will send a message
func (m *Mailbox) Send(msg generic.T, wait bool) (state StateCode) {
	m.mux.Lock()
	if m.isClosed() {
		goto END
	}

	state = m.send(msg, wait)

END:
	m.mux.Unlock()
	return
}

// Batch will send a batch of messages
func (m *Mailbox) Batch(msgs ...generic.T) {
	m.mux.Lock()
	if m.isClosed() {
		goto END
	}

	// Iterate through each message
	for _, msg := range msgs {
		m.send(msg, true)
	}

END:
	m.mux.Unlock()
}

// Receive will receive a message and state (See the "State" constants for more information)
func (m *Mailbox) Receive(wait bool) (msg generic.T, state StateCode) {
	m.mux.Lock()
	msg, state = m.receive(wait)
	m.mux.Unlock()
	return
}

// Listen will return all current and inbound messages until either:
//	- The mailbox is empty and closed
//	- The end boolean is returned
func (m *Mailbox) Listen(fn func(msg generic.T) (end bool)) (state StateCode) {
	var msg generic.T
	m.mux.Lock()
	// Iterate until break is called
	for {
		// Get message and state
		if msg, state = m.receive(true); state != StateOK {
			// Receiving was not successful, break
			break
		}

		// Provide message to provided function
		if fn(msg) {
			// End was returned as true, set state accordingly and break
			state = StateEnded
			break
		}
	}

	m.mux.Unlock()
	return
}

// Close will close a mailbox
func (m *Mailbox) Close() {
	// Attempt to set closed state to 1 (from 0)
	if !atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		// Already closed, return early
		return
	}

	// Notify senders to attempt to send again
	m.sc.Broadcast()
	// Notify receivers to attempty to receive again
	m.rc.Broadcast()
}

// StateCode represents the state of a response
type StateCode uint8

const (
	// StateOK is returned when the request was OK
	StateOK StateCode = iota
	// StateEmpty is returned when the request was empty
	// Note: This will be used when the reject option is implemented
	StateEmpty
	// StateFull is returned when a receiving channel is full and wait is false for sending
	StateFull
	// StateEnded is returned when the client ends a listening
	StateEnded
	// StateClosed is returned when the calling mailbox is closed
	StateClosed
)

// Interface defines the behaviour of a mailbox, it can be implemented
// with a different type of elements.
type Interface interface {
	Send(msg generic.T, wait bool)
	Batch(msgs ...generic.T)
	Receive(wait bool) (msg generic.T, state StateCode)
	Listen(fn func(msg generic.T) (end bool)) (state StateCode)
	Close()
}
