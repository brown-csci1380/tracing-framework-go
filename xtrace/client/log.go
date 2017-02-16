package client

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/brown-csci1380/tracing-framework-go/xtrace/client/internal"
	"github.com/brown-csci1380/tracing-framework-go/xtrace/internal/pubsub"
	"github.com/golang/protobuf/proto"
)

var client *pubsub.Client

var connectOnce sync.Once = sync.Once{}
var disconnectOnce sync.Once = sync.Once{}

var DefaultServerString string = "localhost:5563"

// Connect initializes a connection to the X-Trace
// server. Connect must be called (and must complete
// successfully) before Log can be called.
func Connect(server string) (err error) {
	connectOnce.Do(func() {
		client, err = pubsub.NewClient(server)
		if err != nil {
			client = nil
		}
	})
	return
}

// Disconnect removes the existing connection to the X-Trace server
func Disconnect() {
	disconnectOnce.Do(func() {
		if client != nil {
			client.Close()
			client = nil
		}
	})
}

type xtraceWriter struct{}

func (x xtraceWriter) Write(p []byte) (n int, err error) {
	Log(string(p))
	return len(p), nil
}

func MakeWriter(wrapped ...io.Writer) io.Writer {
	return io.MultiWriter(append(wrapped, xtraceWriter{})...)
}

var topic = []byte("xtrace")
var processName = strings.Join(os.Args, " ")

var pnameOnce = sync.Once{}

func SetProcessName(pname string) {
	pnameOnce.Do(func() {
		processName = pname
	})
}

// Log a given message with the extra preceding events given
// adds a ParentEventId for all precedingEvents _in addition_ to the recorded parent of this event
func LogRedundancies(str string, precedingEvents []int64) {
	if client == nil {
		//fail silently
		return
	}

	parent, event := newEvent()
	var report internal.XTraceReportv4

	report.TaskId = new(int64)
	*report.TaskId = GetTaskID()
	if GetTaskID() <= 0 {
		return
	}
	report.ParentEventId = append(precedingEvents, parent)
	report.EventId = new(int64)
	*report.EventId = event
	report.Label = new(string)
	*report.Label = str

	report.Timestamp = new(int64)
	*report.Timestamp = time.Now().UnixNano() / 1000 // milliseconds

	report.ProcessId = new(int32)
	*report.ProcessId = int32(os.Getpid())
	report.ProcessName = new(string)
	*report.ProcessName = processName
	host, err := os.Hostname()
	if err != nil {
		report.Host = new(string)
		*report.Host = host
	}

	// report.ThreadName = new(string)
	// *report.ThreadName = "Thread name"
	report.Agent = new(string)
	*report.Agent = str

	if getLocal().tags != nil {
		report.Tags = getLocal().tags
		getLocal().tags = nil
	}

	buf, err := proto.Marshal(&report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "internal error: %v", err)
	}

	// NOTE(joshlf): Currently, Log blocks until the log message
	// has been written to the TCP connection to the X-Trace server.
	// This makes testing easier, but ideally we should optimize
	// so that the program can block before it quits, but each
	// call to Log is not blocking.
	client.PublishBlock(topic, buf)
}

// Log logs the given message. Log must not be
// called before Connect has been called successfully.
func Log(str string) {
	LogRedundancies(str, PopRedundancies())
}

func Logf(format string, args ...interface{}) {
	Log(fmt.Sprintf(format, args...))
}
