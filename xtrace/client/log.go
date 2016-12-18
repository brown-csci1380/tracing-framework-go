package client

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brownsys/tracing-framework-go/xtrace/client/internal"
	"github.com/brownsys/tracing-framework-go/xtrace/internal/pubsub"
	"github.com/golang/protobuf/proto"
)

var client *pubsub.Client

var defaultLogLocation *log.Logger

// Connect initializes a connection to the X-Trace
// server. Connect must be called (and must complete
// successfully) before Log can be called.
func Connect(server string) error {
	var err error
	client, err = pubsub.NewClient(server)
	return err
}

var topic = []byte("xtrace")
var processName = strings.Join(os.Args, " ")

func SetFallbackLogger(l *log.Logger) {
	defaultLogLocation = l
}

func SetProcessName(pname string) {
	processName = pname
}

// Log a given message with the extra preceding events given
// adds a ParentEventId for all precedingEvents _in addition_ to the recorded parent of this event
func LogRedundancies(str string, precedingEvents ...int64) {
	if client == nil {
		if defaultLogLocation != nil {
			// if given a default location, log to there
			defaultLogLocation.Println("xtrace (task:", GetTaskID(), "): Logged with no connection:", str, "events:", precedingEvents)
		}
		//else fail silently
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
	LogRedundancies(str, PopRedundancies()...)
}

func Logf(format string, args ...interface{}) {
	Log(fmt.Sprintf(format, args...))
}
