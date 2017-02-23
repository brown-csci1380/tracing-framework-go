package client

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/brown-csci1380/tracing-framework-go/local"
)

var token local.Token

type localStorage struct {
	taskID       int64
	eventID      int64
	redundancies []int64
	tags         []string
}

// exported type for RPC calls
type RPCMetadata struct {
	TaskID int64
	Events []int64
}

func init() {
	token = local.Register(&localStorage{
		taskID:       randInt64(),
		eventID:      randInt64(),
		redundancies: []int64{},
	}, local.Callbacks{
		func(l interface{}) interface{} {
			// deep copy l
			n := *(l.(*localStorage))
			n.redundancies = []int64{}
			return &n
		},
	})
}

// Runs the given function in a new goroutine, but copies the
// local vars from the current goroutine first.
func XGo(f func()) {
	go func(f1 func(), f2 func()) {
		f1()
		f2()
	}(local.GetSpawnCallback(), f)
}

func getLocal() *localStorage {
	return local.GetLocal(token).(*localStorage)
}

func GetRPCMetadata() RPCMetadata {
	l := getLocal()
	var r RPCMetadata
	r.Events = append(l.redundancies, l.eventID)
	r.TaskID = l.taskID
	return r
}

func (r *RPCMetadata) Set() {
	l := getLocal()
	r.Events = append(l.redundancies, l.eventID)
	r.TaskID = l.taskID
}

func RPCReceived(r RPCMetadata, msg string) {
	SetTaskID(r.TaskID)
	events := r.Events
	if len(events) >= 1 {
		SetEventID(events[0])
		getLocal().redundancies = append([]int64{}, events[1:]...)
	}
	Log(msg)
}

func RPCReturned(r RPCMetadata, msg string) {
	SetTaskID(r.TaskID)
	AddRedundancies(r.Events...)
	Log(msg)
}

// SetEventID sets the current goroutine's X-Trace Event ID.
// This should be used when propagating Event IDs over RPC
// calls or other channels.
//
// WARNING: This will overwrite any previous Event ID,
// so call with caution.
func SetEventID(eventID int64) {
	getLocal().eventID = eventID
}

// SetTaskID sets the current goroutine's X-Trace Task ID.
// This should be used when propagating Task IDs over RPC
// calls or other channels.
//
// WARNING: This will overwrite any previous Task ID,
// so call with caution.
func SetTaskID(taskID int64) {
	getLocal().taskID = taskID
}

func AddTags(str ...string) {
	if getLocal().tags == nil {
		getLocal().tags = str
	} else {
		getLocal().tags = append(getLocal().tags, str...)
	}
}

func NewTask(tags ...string) {
	SetTaskID(randInt64())
	SetEventID(randInt64())
	getLocal().tags = tags
}

// GetEventID gets the current goroutine's X-Trace Event ID.
// Note that if one has not been set yet, GetEventID will
// return 0. This should be used when propagating Event IDs
// over RPC calls or other channels.
func GetEventID() (eventID int64) {
	return getLocal().eventID
}

// GetTaskID gets the current goroutine's X-Trace Task ID.
// Note that if one has not been set yet, GetTaskID will
// return 0. This should be used when propagating Task IDs
// over RPC calls or other channels.
func GetTaskID() (taskID int64) {
	return getLocal().taskID
}

func AddRedundancies(eventIDs ...int64) {
	getLocal().redundancies = append(getLocal().redundancies, eventIDs...)
}

func PopRedundancies() []int64 {
	eventIDs := append([]int64{}, getLocal().redundancies...)
	getLocal().redundancies = []int64{}
	return eventIDs
}

func newEvent() (parent, event int64) {
	parent = GetEventID()
	event = randInt64()
	SetEventID(event)
	return parent, event
}

func randInt64() int64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(fmt.Errorf("could not read random bytes: %v", err))
	}
	// shift to guarantee high bit is 0 and thus
	// int64 version is non-negative
	return int64(binary.BigEndian.Uint64(b[:]) >> 1)
}
