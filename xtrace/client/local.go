package client

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/brownsys/tracing-framework-go/local"
	"strconv"
	"strings"
)

var token local.Token

const EVENT_KEY = "events"
const TASK_KEY = "task"
const MD_KEY = "xtr_metadata"

type localStorage struct {
	taskID       int64
	eventID      int64
	redundancies []int64
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
			return &n
		},
	})
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

func formatIDs(event int64, ids []int64) []string {
	list := make([]string, len(ids)+1)
	list[0] = strconv.FormatInt(event, 10)
	for idx, val := range ids {
		list[idx+1] = strconv.FormatInt(val, 10)
	}
	return list
}

func getIDs(ids string) []int64 {
	id_strings := strings.Split(ids, ",")
	list := make([]int64, len(id_strings))
	for idx, val := range id_strings {
		list[idx], _ = strconv.ParseInt(val, 10, 64)
	}
	return list
}

// Returns a slice of strings suitable for passing to grpc/metadata.Pairs
func GRPCMetadata() []string {
	l := getLocal()
	return []string{TASK_KEY, strconv.FormatInt(l.taskID, 10), EVENT_KEY, strings.Join(formatIDs(l.eventID, l.redundancies), ",")}
}

func GRPCRecieved(md map[string][]string, msg string) {
	event_strs, ok := md[EVENT_KEY]
	if !ok || len(event_strs) < 1 {
		fmt.Printf("bad metadata: %v\n", md)
		return
	}
	events := getIDs(md[EVENT_KEY][0])

	task_strs, ok := md[TASK_KEY]
	if !ok || len(task_strs) < 1 {
		fmt.Printf("bad metadata: %v\n", md)
		return
	}
	taskID, err := strconv.ParseInt(task_strs[0], 10, 64)
	if err != nil {
		fmt.Printf("bad metadata: %v\n", md)
		return
	}

	RPCReceived(RPCMetadata{
		TaskID: taskID,
		Events: events,
	}, msg)
}

func GRPCReturned(md map[string][]string, msg string) {
	event_strs, ok := md[EVENT_KEY]
	if !ok || len(event_strs) < 1 {
		fmt.Printf("bad metadata: %v\n", md)
		return
	}
	events := getIDs(md[EVENT_KEY][0])

	task_strs, ok := md[TASK_KEY]
	if !ok || len(task_strs) < 1 {
		fmt.Printf("bad metadata: %v\n", md)
		return
	}
	taskID, err := strconv.ParseInt(task_strs[0], 10, 64)
	if err != nil {
		fmt.Printf("bad metadata: %v\n", md)
		return
	}

	RPCReturned(RPCMetadata{
		TaskID: taskID,
		Events: events,
	}, msg)
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

func NewTask() {
	SetTaskID(randInt64())
	SetEventID(randInt64())
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
