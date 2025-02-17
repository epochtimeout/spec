// Copyright 2024 Ivan Korobkov. All rights reserved.
// Use of this software is governed by the MIT License
// that can be found in the LICENSE file.

package mpx

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/basecomplextech/baselibrary/alloc/bytequeue"
	"github.com/basecomplextech/baselibrary/async"
	"github.com/basecomplextech/baselibrary/bin"
	"github.com/basecomplextech/baselibrary/pools"
	"github.com/basecomplextech/baselibrary/status"
	"github.com/basecomplextech/spec/proto/pmpx"
)

type channelState struct {
	id   bin.Bin128
	ctx  *context
	conn internalConn

	client     bool  // client or server
	initWindow int32 // initial window size

	opened     atomic.Bool
	closed     atomic.Bool
	closedUser atomic.Bool // close user once

	sendMu         sync.Mutex    // enforce single sender
	sendWindow     atomic.Int32  // remaining send window, can become negative on sending large messages
	sendWindowWait chan struct{} // wait for send window increment
	sender         channelSender

	recvQueue bytequeue.Queue // data queue
	recvBytes atomic.Int32    // number of received byte, sent as window delta when >= initWindow/2
}

func newChannelState(conn internalConn, client bool, id bin.Bin128, window int32) *channelState {
	s := channelStatePool.New()
	s.id = id
	s.ctx = newContext(conn)
	s.conn = conn

	s.client = client
	s.initWindow = window

	s.sender = newChanSender(s, conn)
	s.sendWindow.Store(window)
	return s
}

func openChannelState(conn internalConn, client bool, msg pmpx.ChannelOpen) *channelState {
	id := msg.Id()
	window := msg.Window()

	s := channelStatePool.New()
	s.id = id
	s.ctx = newContext(conn)
	s.conn = conn

	s.client = client
	s.initWindow = window
	s.opened.Store(true)

	s.sender = newChanSender(s, conn)
	s.sendWindow.Store(window)
	return s
}

// open/close

func (s *channelState) open() {
	if s.opened.Load() {
		return
	}
	if !s.client {
		panic("cannot open server channel")
	}

	s.opened.Store(true)
}

func (s *channelState) close() {
	// Try to close channel
	ok := s.closed.CompareAndSwap(false, true)
	if !ok {
		return
	}

	// Cancel context, close receive queue
	s.ctx.Cancel()
	s.recvQueue.Close()
}

// receive

func (s *channelState) receiveMessage(msg pmpx.Message) status.Status {
	code := msg.Code()

	switch code {
	case pmpx.Code_ChannelClose:
		return s.receiveClose(msg.ChannelClose())
	case pmpx.Code_ChannelData:
		return s.receiveData(msg.ChannelData())
	case pmpx.Code_ChannelWindow:
		return s.receiveWindow(msg.ChannelWindow())
	}

	return mpxErrorf("message must be handled by connection, code=%v", code)
}

func (s *channelState) receiveClose(msg pmpx.ChannelClose) status.Status {
	if s.closed.Load() {
		return status.OK
	}

	// Maybe receive data
	data := msg.Data()
	if len(data) > 0 {
		_, _ = s.recvQueue.Write(data) // ignore end and false, receive queues are unbounded
	}

	// Close channel
	s.close()
	return status.OK
}

func (s *channelState) receiveData(msg pmpx.ChannelData) status.Status {
	data := msg.Data()
	_, _ = s.recvQueue.Write(data) // ignore end and false, receive queues are unbounded
	return status.OK
}

func (s *channelState) receiveWindow(msg pmpx.ChannelWindow) status.Status {
	// Increment send window
	delta := msg.Delta()
	s.sendWindow.Add(delta)

	// Notify waiters
	select {
	case s.sendWindowWait <- struct{}{}:
	default:
	}
	return status.OK
}

// window

func (s *channelState) decrementSendWindow(ctx async.Context, data []byte) status.Status {
	// Check data size
	n := len(data)
	if n > math.MaxInt32 {
		return mpxErrorf("message too large, size=%d", n)
	}
	size := int32(n)

	for {
		// Decrement send window for normal small messages
		window := s.sendWindow.Load()
		if window >= int32(size) {
			s.sendWindow.Add(-size)
			return status.OK
		}

		// Decrement send window for large messages, when the remaining window
		// is greater than the half of the initial window, but the message size
		// still exceeds it.
		if window >= s.initWindow/2 {
			s.sendWindow.Add(-size)
			return status.OK
		}

		// Wait for send window increment
		select {
		case <-ctx.Wait():
			return ctx.Status()
		case <-s.ctx.Wait():
			return statusChannelClosed
		case <-s.sendWindowWait:
		}
	}
}

// reset

func (s *channelState) reset() {
	// Free context
	if s.ctx != nil {
		s.ctx.Free()
		s.ctx = nil
	}

	// Clear wait channel
	sendWindowWait := s.sendWindowWait
	select {
	case <-sendWindowWait:
	default:
	}

	// Reset receive queue
	recvQueue := s.recvQueue
	recvQueue.Reset()

	// Reset state
	*s = channelState{}
	s.sendWindowWait = sendWindowWait
	s.recvQueue = recvQueue
}

// pool

var channelStatePool = pools.NewPoolFunc(
	func() *channelState {
		return &channelState{
			sendWindowWait: make(chan struct{}, 1),
			recvQueue:      bytequeue.New(),
		}
	},
)

func releaseChannelState2(s *channelState) {
	s.reset()
	channelStatePool.Put(s)
}
