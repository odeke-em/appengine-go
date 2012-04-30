// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

/*
Package taskqueue provides a client for App Engine's taskqueue service.
Using this service, applications may perform work outside a user's request.

A Task may be constucted manually; alternatively, since the most common
taskqueue operation is to add a single POST task, NewPOSTTask makes it easy.

	t := taskqueue.NewPOSTTask("/worker", url.Values{
		"key": {key},
	})
	taskqueue.Add(c, t, "") // add t to the default queue
*/
package taskqueue

// TODO: Bulk task deleting, queue management.

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"appengine"
	"appengine_internal"
	taskqueue_proto "appengine_internal/taskqueue"
	"code.google.com/p/goprotobuf/proto"
)

// RetryOptions let you control whether to retry a task and the backoff intervals between tries.
type RetryOptions struct {
	// Number of tries/leases after which the task fails permanently and is deleted.
	// If AgeLimit is also set, both limits must be exceeded for the task to fail permanently.
	RetryLimit int32

	// Maximum time allowed since the task's first try before the task fails permanently and is deleted (only for push tasks).
	// If RetryLimit is also set, both limits must be exceeded for the task to fail permanently.
	AgeLimit time.Duration

	// Minimum time between successive tries (only for push tasks).
	MinBackoff time.Duration

	// Maximum time between successive tries (only for push tasks).
	MaxBackoff time.Duration

	// Maximum number of times to double the interval between successive tries before the intervals increase linearly (only for push tasks).
	MaxDoublings int32

	// If MaxDoublings is zero, set ApplyZeroMaxDoublings to true to override the default non-zero value.
	// Otherwise a zero MaxDoublings is ignored and the default is used.
	ApplyZeroMaxDoublings bool
}

// toRetryParameter converts RetryOptions to taskqueue_proto.TaskQueueRetryParameters.
func (opt *RetryOptions) toRetryParameters() *taskqueue_proto.TaskQueueRetryParameters {
	params := &taskqueue_proto.TaskQueueRetryParameters{}
	if opt.RetryLimit > 0 {
		params.RetryLimit = proto.Int32(opt.RetryLimit)
	}
	if opt.AgeLimit > 0 {
		params.AgeLimitSec = proto.Int64(int64(opt.AgeLimit.Seconds()))
	}
	if opt.MinBackoff > 0 {
		params.MinBackoffSec = proto.Float64(opt.MinBackoff.Seconds())
	}
	if opt.MaxBackoff > 0 {
		params.MaxBackoffSec = proto.Float64(opt.MaxBackoff.Seconds())
	}
	if opt.MaxDoublings > 0 || (opt.MaxDoublings == 0 && opt.ApplyZeroMaxDoublings) {
		params.MaxDoublings = proto.Int32(opt.MaxDoublings)
	}
	return params
}

// A Task represents a task to be executed.
type Task struct {
	// Path is the worker URL for the task.
	// If unset, it will default to /_ah/queue/<queue_name>.
	Path string

	// Payload is the data for the task.
	// This will be delivered as the HTTP request body.
	// It is only used when Method is POST, PUT or PULL.
	// url.Values' Encode method may be used to generate this for POST requests.
	Payload []byte

	// Additional HTTP headers to pass at the task's execution time.
	// To schedule the task to be run with an alternate app version
	// or backend, set the "Host" header.
	Header http.Header

	// Method is the HTTP method for the task ("GET", "POST", etc.),
	// or "PULL" if this is task is destined for a pull-based queue.
	// If empty, this defaults to "POST".
	Method string

	// A name for the task.
	// If empty, a name will be chosen.
	Name string

	// Delay is how far into the future this task should execute.
	Delay time.Duration

	// Time before which the task may neither be executed or leased.
	// Set on tasks returned from Lease calls.
	ETA time.Time

	// The number of times the task has been dispatched or leased.
	RetryCount int32

	// Tag for the task. Only used when Method is PULL.
	Tag string

	// Retry options for this task. May be nil.
	RetryOptions *RetryOptions
}

func (t *Task) method() string {
	if t.Method == "" {
		return "POST"
	}
	return t.Method
}

// NewPOSTTask creates a Task that will POST to a path with the given form data.
func NewPOSTTask(path string, params url.Values) *Task {
	h := make(http.Header)
	h.Set("Content-Type", "application/x-www-form-urlencoded")
	return &Task{
		Path:    path,
		Payload: []byte(params.Encode()),
		Header:  h,
		Method:  "POST",
	}
}

func newAddReq(task *Task, queueName string) (*taskqueue_proto.TaskQueueAddRequest, error) {
	if queueName == "" {
		queueName = "default"
	}
	req := &taskqueue_proto.TaskQueueAddRequest{
		QueueName: []byte(queueName),
		TaskName:  []byte(task.Name),
		EtaUsec:   proto.Int64(time.Now().Add(task.Delay).UnixNano() / 1e3),
	}
	method := task.method()
	if method == "PULL" {
		// Pull-based task
		req.Body = task.Payload
		req.Mode = taskqueue_proto.NewTaskQueueMode_Mode(taskqueue_proto.TaskQueueMode_PULL)
		if task.Tag != "" {
			req.Tag = []byte(task.Tag)
		}
		// TODO: More fields will need to be set.
	} else {
		// HTTP-based task
		if v, ok := taskqueue_proto.TaskQueueAddRequest_RequestMethod_value[method]; ok {
			req.Method = taskqueue_proto.NewTaskQueueAddRequest_RequestMethod(
				taskqueue_proto.TaskQueueAddRequest_RequestMethod(v))
		} else {
			return nil, fmt.Errorf("taskqueue: bad method %q", method)
		}
		req.Url = []byte(task.Path)
		for k, vs := range task.Header {
			for _, v := range vs {
				req.Header = append(req.Header, &taskqueue_proto.TaskQueueAddRequest_Header{
					Key:   []byte(k),
					Value: []byte(v),
				})
			}
		}
		if method == "POST" || method == "PUT" {
			req.Body = task.Payload
		}
	}

	if task.RetryOptions != nil {
		req.RetryParameters = task.RetryOptions.toRetryParameters()
	}

	return req, nil
}

// Add adds the task to a named queue.
// An empty queue name means that the default queue will be used.
// Add returns an equivalent Task with defaults filled in, including setting
// the task's Name field to the chosen name if the original was empty.
func Add(c appengine.Context, task *Task, queueName string) (*Task, error) {
	req, err := newAddReq(task, queueName)
	if err != nil {
		return nil, err
	}
	res := &taskqueue_proto.TaskQueueAddResponse{}
	if err := c.Call("taskqueue", "Add", req, res, nil); err != nil {
		return nil, err
	}
	resultTask := *task
	resultTask.Method = task.method()
	if task.Name == "" {
		resultTask.Name = string(res.ChosenTaskName)
	}
	return &resultTask, nil
}

// AddMulti adds multiple tasks to a named queue.
// An empty queue name means that the default queue will be used.
// AddMulti returns a slice of equivalent tasks with defaults filled in, including setting
// each task's Name field to the chosen name if the original was empty.
// If a given task is badly formed or could not be added, an appengine.MultiError is returned.
func AddMulti(c appengine.Context, tasks []*Task, queueName string) ([]*Task, error) {
	req := &taskqueue_proto.TaskQueueBulkAddRequest{
		AddRequest: make([]*taskqueue_proto.TaskQueueAddRequest, len(tasks)),
	}
	me, any := make(appengine.MultiError, len(tasks)), false
	for i, t := range tasks {
		req.AddRequest[i], me[i] = newAddReq(t, queueName)
		any = any || me[i] != nil
	}
	if any {
		return nil, me
	}
	res := &taskqueue_proto.TaskQueueBulkAddResponse{}
	if err := c.Call("taskqueue", "BulkAdd", req, res, nil); err != nil {
		return nil, err
	}
	if len(res.Taskresult) != len(tasks) {
		return nil, errors.New("taskqueue: server error")
	}
	tasksOut := make([]*Task, len(tasks))
	for i, tr := range res.Taskresult {
		tasksOut[i] = new(Task)
		*tasksOut[i] = *tasks[i]
		tasksOut[i].Method = tasksOut[i].method()
		if tasksOut[i].Name == "" {
			tasksOut[i].Name = string(tr.ChosenTaskName)
		}
		if *tr.Result != taskqueue_proto.TaskQueueServiceError_OK {
			me[i] = &appengine_internal.APIError{
				Service: "taskqueue",
				Code:    int32(*tr.Result),
			}
			any = true
		}
	}
	if any {
		return tasksOut, me
	}
	return tasksOut, nil
}

// Delete deletes a task from a named queue.
func Delete(c appengine.Context, task *Task, queueName string) error {
	req := &taskqueue_proto.TaskQueueDeleteRequest{
		QueueName: []byte(queueName),
		TaskName:  [][]byte{[]byte(task.Name)},
	}
	res := &taskqueue_proto.TaskQueueDeleteResponse{}
	if err := c.Call("taskqueue", "Delete", req, res, nil); err != nil {
		return err
	}
	for _, ec := range res.Result {
		if ec != taskqueue_proto.TaskQueueServiceError_OK {
			return &appengine_internal.APIError{
				Service: "taskqueue",
				Code:    int32(ec),
			}
		}
	}
	return nil
}

func lease(c appengine.Context, maxTasks int, queueName string, leaseTime int, groupByTag bool, tag []byte) ([]*Task, error) {
	req := &taskqueue_proto.TaskQueueQueryAndOwnTasksRequest{
		QueueName:    []byte(queueName),
		LeaseSeconds: proto.Float64(float64(leaseTime)),
		MaxTasks:     proto.Int64(int64(maxTasks)),
		GroupByTag:   proto.Bool(groupByTag),
		Tag:          tag,
	}
	res := &taskqueue_proto.TaskQueueQueryAndOwnTasksResponse{}
	if err := c.Call("taskqueue", "QueryAndOwnTasks", req, res, nil); err != nil {
		return nil, err
	}
	tasks := make([]*Task, len(res.Task))
	for i, t := range res.Task {
		tasks[i] = &Task{
			Payload:    t.Body,
			Name:       string(t.TaskName),
			Method:     "PULL",
			ETA:        time.Unix(0, *t.EtaUsec*1e3),
			RetryCount: *t.RetryCount,
			Tag:        string(t.Tag),
		}
	}
	return tasks, nil
}

// Lease leases tasks from a queue.
// leaseTime is in seconds.
// The number of tasks fetched will be at most maxTasks.
func Lease(c appengine.Context, maxTasks int, queueName string, leaseTime int) ([]*Task, error) {
	return lease(c, maxTasks, queueName, leaseTime, false, nil)
}

// LeaseByTag leases tasks from a queue, grouped by tag.
// If tag is nil, then the returned tasks are grouped by the tag of the task with earliest Eta.
// leaseTime is in seconds.
// The number of tasks fetched will be at most maxTasks.
func LeaseByTag(c appengine.Context, maxTasks int, queueName string, leaseTime int, tag string) ([]*Task, error) {
	return lease(c, maxTasks, queueName, leaseTime, true, []byte(tag))
}

// Purge removes all tasks from a queue.
func Purge(c appengine.Context, queueName string) error {
	req := &taskqueue_proto.TaskQueuePurgeQueueRequest{
		QueueName: []byte(queueName),
	}
	res := &taskqueue_proto.TaskQueuePurgeQueueResponse{}
	return c.Call("taskqueue", "PurgeQueue", req, res, nil)
}

// ModifyLease modifies the lease of a task.
// Used to request more processing time, or to abandon processing.
// leaseTime is in seconds and must not be negative.
func ModifyLease(c appengine.Context, task *Task, queueName string, leaseTime int) error {
	req := &taskqueue_proto.TaskQueueModifyTaskLeaseRequest{
		QueueName:    []byte(queueName),
		TaskName:     []byte(task.Name),
		EtaUsec:      proto.Int64(task.ETA.UnixNano() / 1e3), // Used to verify ownership.
		LeaseSeconds: proto.Float64(float64(leaseTime)),
	}
	res := &taskqueue_proto.TaskQueueModifyTaskLeaseResponse{}
	if err := c.Call("taskqueue", "ModifyTaskLease", req, res, nil); err != nil {
		return err
	}
	task.ETA = time.Unix(0, *res.UpdatedEtaUsec*1e3)
	return nil
}

func init() {
	appengine_internal.RegisterErrorCodeMap("taskqueue", taskqueue_proto.TaskQueueServiceError_ErrorCode_name)
}
