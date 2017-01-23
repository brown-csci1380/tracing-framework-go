package grpcutil

import (
	"github.com/brown-csci1380/tracing-framework-go/xtrace/client"
	"strconv"
	"strings"
)

const EVENT_KEY = "events"
const TASK_KEY = "task"
const MD_KEY = "xtr_metadata"

func formatIDs(ids []int64) []string {
	list := make([]string, len(ids))
	for idx, val := range ids {
		list[idx] = strconv.FormatInt(val, 10)
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
	r := client.GetRPCMetadata()
	return []string{TASK_KEY, strconv.FormatInt(r.TaskID, 10), EVENT_KEY, strings.Join(formatIDs(r.Events), ",")}
}

func GRPCRecieved(md map[string][]string, msg string) {
	event_strs, ok := md[EVENT_KEY]
	if !ok || len(event_strs) < 1 {
		client.Log(msg)
		return
	}
	events := getIDs(md[EVENT_KEY][0])

	task_strs, ok := md[TASK_KEY]
	if !ok || len(task_strs) < 1 {
		client.Log(msg)
		return
	}
	taskID, err := strconv.ParseInt(task_strs[0], 10, 64)
	if err != nil {
		client.Log(msg)
		return
	}

	client.RPCReceived(client.RPCMetadata{
		TaskID: taskID,
		Events: events,
	}, msg)
}

func GRPCReturned(md map[string][]string, msg string) {
	event_strs, ok := md[EVENT_KEY]
	if !ok || len(event_strs) < 1 {
		client.Log(msg)
		return
	}
	events := getIDs(md[EVENT_KEY][0])

	task_strs, ok := md[TASK_KEY]
	if !ok || len(task_strs) < 1 {
		client.Log(msg)
		return
	}
	taskID, err := strconv.ParseInt(task_strs[0], 10, 64)
	if err != nil {
		client.Log(msg)
		return
	}

	client.RPCReturned(client.RPCMetadata{
		TaskID: taskID,
		Events: events,
	}, msg)
}
