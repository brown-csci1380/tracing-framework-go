package client

import (
	"fmt"
	"sync"
)

var registeredChannels map[interface{}]chan int64 = make(map[interface{}]chan int64)
var rcLock *sync.RWMutex = &sync.RWMutex{}

const BUF = 2

// called by the reciever of a value across a channel to find out what event sent the value
// the argument 'channel' should be any channel the goroutine is waiting to recieve a value
// from. returns a chan int64 which will emit every eventid that has registered as sending
// along the channel
func RegisterChannelReciever(channel interface{}) (ch chan int64) {
	rcLock.RLock()
	ch, ok := registeredChannels[channel]
	rcLock.RUnlock()
	if !ok {
		ch = make(chan int64, BUF)
		rcLock.Lock()
		registeredChannels[channel] = ch
		rcLock.Unlock()
	}
	return
}

// Convenience method. Get the event id of the last sender in the channel, and
// add it to the local store of redundant edges
func AddChannelEvent(channel interface{}) {
	redund := GetChannelSender(channel)
	fmt.Printf("Found redundancy: %v\n", redund)
	AddRedundancies(redund...)
}

// Get the last EventID that sent a value along the provided channel.
// Returns a singleton slice containing the most recent EventID of the sender,
// so long as the sender called called RegisterChannelSender before sending
// the value. Returns an empty slice if no sender is known.
func GetChannelSender(channel interface{}) []int64 {
	ch := RegisterChannelReciever(channel)

	select {
	// if the recv blocks, there must have been no call to RegisterChannelSender on the
	// channel as the function argument
	case sender := <-ch:
		return []int64{sender}
	default:
		return []int64{}
	}
}

// called by the sender of a value over a channel. Informs the future recipient of the value
// which event ID it originated from
func RegisterChannelSender(channel interface{}) {
	rcLock.RLock()
	ch, ok := registeredChannels[channel]
	rcLock.RUnlock()

	if !ok {
		ch = make(chan int64, BUF)
		rcLock.Lock()
		registeredChannels[channel] = ch
		rcLock.Unlock()
	}
	eventID := GetEventID()

	select {
	case ch <- eventID:
		// sent immediately
		return
	default:
		// do this in a separate goroutine because chan sends can block unless there is a reciever ready
		go func() {
			ch <- eventID
		}()
	}

}
