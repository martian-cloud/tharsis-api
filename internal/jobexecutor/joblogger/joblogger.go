package joblogger

//go:generate mockery --name JobLogger --inpackage --case underscore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	defaultMaxBytesPerPatch = 1024 * 1024 // in bytes
	defaultUpdateInterval   = 3 * time.Second
)

// Logger is an interface for logging job output
type Logger interface {
	// Close flushes the logger
	Close()
	// Infof writes an info log to the job's log output
	Infof(format string, a ...interface{})
	// Errorf writes an error log to the job's log output
	Errorf(format string, a ...interface{})
	// Write will append the data to the log buffer
	Write(data []byte) (n int, err error)
	// Start starts the logger
	Start()
	// Flush flushes the logger
	Flush()
}

var _ Logger = (*jobLogger)(nil)

type jobLogger struct {
	sentTime         time.Time
	client           jobclient.Client
	logger           logger.Logger
	buffer           *LogBuffer
	finished         chan bool
	jobID            string
	bytesSent        int
	updateInterval   time.Duration
	maxBytesPerPatch int
	lock             sync.RWMutex
}

// NewLogger creates a new Logger
func NewLogger(jobID string, client jobclient.Client, logger logger.Logger) (Logger, error) {
	buffer, err := NewLogBuffer()
	if err != nil {
		return nil, err
	}

	return &jobLogger{
		jobID:            jobID,
		buffer:           buffer,
		maxBytesPerPatch: defaultMaxBytesPerPatch,
		updateInterval:   defaultUpdateInterval,
		client:           client,
		logger:           logger,
	}, nil
}

// Close flushes the logger
func (j *jobLogger) Close() {
	j.finish()
}

// Infof writes an info log to the job's log output
func (j *jobLogger) Infof(format string, a ...interface{}) {
	j.Write([]byte(fmt.Sprintf(AnsiBoldCyan+format+"\n"+AnsiReset, a...)))
}

// Errorf writes an error log to the job's log output
func (j *jobLogger) Errorf(format string, a ...interface{}) {
	j.Write([]byte(fmt.Sprintf(AnsiBoldRed+format+"\n"+AnsiReset, a...)))
}

// Write will append the data to the log buffer
func (j *jobLogger) Write(data []byte) (n int, err error) {
	j.logger.Infof("JOB OUTPUT: %s", string(data))
	return j.buffer.Write(data)
}

// nolint:unused
func (j *jobLogger) checksum() string {
	return j.buffer.Checksum()
}

// nolint:unused
func (j *jobLogger) bytesize() int {
	return j.buffer.Size()
}

func (j *jobLogger) Start() {
	j.finished = make(chan bool)
	go j.run()
}

func (j *jobLogger) Flush() {
	for j.anyLogsToSend() {
		if err := j.sendPatch(); err != nil {
			j.logger.Errorf("Failed to send logs %v", err)
			time.Sleep(10 * time.Second)
		}
	}
}

func (j *jobLogger) finish() {
	j.finished <- true
	j.Flush()
	j.buffer.Close()
}

func (j *jobLogger) anyLogsToSend() bool {
	j.lock.RLock()
	defer j.lock.RUnlock()

	return j.buffer.Size() != j.bytesSent
}

func (j *jobLogger) sendPatch() error {
	j.lock.RLock()
	content, err := j.buffer.Bytes(j.bytesSent, j.maxBytesPerPatch)
	bytesSent := j.bytesSent
	j.lock.RUnlock()

	if err != nil {
		return err
	}

	if len(content) == 0 {
		return nil
	}

	if err := j.client.SaveJobLogs(context.Background(), j.jobID, j.bytesSent, content); err != nil {
		return err
	}

	j.lock.Lock()
	j.sentTime = time.Now()
	j.bytesSent = bytesSent + len(content)
	j.lock.Unlock()

	return nil
}

func (j *jobLogger) run() {
	for {
		select {
		case <-time.After(j.updateInterval):
			if err := j.sendPatch(); err != nil {
				j.logger.Errorf("Failed to send log patch %v", err)
			}
		case <-j.finished:
			return
		}
	}
}
