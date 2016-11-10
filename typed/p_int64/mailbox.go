package mailbox

import (
	"sync"
	"sync/atomic"
)

// New returns a new instance of Mailbox
func New(sz int) *Mailbox {
	mb := Mailbox{
		cap:  sz,
		tail: -1,

		s: make([]*int64, sz),
	}

	// Initialize the conds
	mb.sc = sync.NewCond(&mb.mux)
	mb.rc = sync.NewCond(&mb.mux)
	return &mb
}

// MailboxIface defines the behaviour of a mailbox, it can be implemented
// with a different type of elements.
type MailboxIface interface {
	Send(msg *int64)
	Batch(msgs ...*int64)
	Receive() (msg *int64, state StateCode)
	Listen(fn func(msg *int64) (end bool)) (state StateCode)
	Close()
}

// Mailbox is used to send and receive messages
type Mailbox struct {
	mux sync.Mutex
	sc  *sync.Cond
	rc  *sync.Cond

	s []*int64

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
func (m *Mailbox) rWait() (ok bool) {
START:
	if m.len > 0 {
		// We have at least one unread message, return true
		return true
	}

	if m.isClosed() {
		// We have an empty inbox AND we are closed, done bro - done.
		return false
	}

	// Let's wait for a signal..
	m.rc.Wait()
	// Signal received, let's check again!
	goto START
}

var empty *int64

// receive is the internal function for receiving messages
func (m *Mailbox) receive() (msg *int64, state StateCode) {
	if !m.rWait() {
		// Ok was returned as false, set state to closed and return
		state = StateClosed
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

// send is the internal function used for sending messages
func (m *Mailbox) send(msg *int64) {
CHECKFREE:
	if m.cap-m.len == 0 {
		// There are no vacant spots in the inbox, time to wait
		m.sc.Wait()
		// We received a signal, check again!
		goto CHECKFREE
	}

	// Goto the next index
	if m.tail++; m.tail == m.cap {
		// Our increment falls out of the bounds of our internal slice, reset to 0
		m.tail = 0
	}

	// Send the new tail as the provided message
	m.s[m.tail] = msg

	// Increment the length
	if m.len++; m.len == 1 {
		// Notify the receivers that we new message
		m.rc.Broadcast()
	}
}

// Send will send a message
func (m *Mailbox) Send(msg *int64) {
	m.mux.Lock()
	if m.isClosed() {
		goto END
	}

	m.send(msg)

END:
	m.mux.Unlock()
}

// Batch will send a batch of messages
func (m *Mailbox) Batch(msgs ...*int64) {
	m.mux.Lock()
	if m.isClosed() {
		goto END
	}

	// Iterate through each message
	for _, msg := range msgs {
		m.send(msg)
	}

END:
	m.mux.Unlock()
}

// Receive will receive a message and state (See the "State" constants for more information)
func (m *Mailbox) Receive() (msg *int64, state StateCode) {
	m.mux.Lock()
	msg, state = m.receive()
	m.mux.Unlock()
	return
}

// Listen will return all current and inbound messages until either:
//	- The mailbox is empty and closed
//	- The end boolean is returned
func (m *Mailbox) Listen(fn func(msg *int64) (end bool)) (state StateCode) {
	var msg *int64
	m.mux.Lock()
	// Iterate until break is called
	for {
		// Get message and state
		if msg, state = m.receive(); state != StateOK {
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
	// StateEnded is returned when the client ends a listening
	StateEnded
	// StateClosed is returned when the calling mailbox is closed
	StateClosed
)
