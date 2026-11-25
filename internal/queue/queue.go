package queue

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/mittalsukhansh/forge/internal/engine"
	"github.com/mittalsukhansh/forge/internal/pipeline"
	"github.com/mittalsukhansh/forge/internal/store"
)

var ErrQueueFull = errors.New("job queue is full")

type Task struct {
	ID       int64
	Pipeline *pipeline.Pipeline
}

type Queue struct {
	tasks  chan Task
	store  *store.Store
	engine *engine.Engine
	ctx    context.Context
	mu     sync.Mutex
	wg     sync.WaitGroup
}

type logWriter struct {
	store *store.Store
	id    int64
}

func (w *logWriter) Write(p []byte) (int, error) {
	err := w.store.AppendLogs(w.id, string(p))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func New(
	ctx context.Context,
	s *store.Store,
	workers int,
) *Queue {
	q := &Queue{
		tasks:  make(chan Task, 32),
		store:  s,
		engine: engine.New(),
		ctx:    ctx,
	}

	for i := 1; i <= workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	return q
}

func (q *Queue) Submit(p *pipeline.Pipeline) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.ctx.Err(); err != nil {
		return 0, err
	}

	if len(q.tasks) == cap(q.tasks) {
		return 0, ErrQueueFull
	}

	id, err := q.store.Create(p.Name)
	if err != nil {
		return 0, err
	}

	q.tasks <- Task{
		ID:       id,
		Pipeline: p,
	}

	log.Printf("JOB_QUEUED id=%d", id)

	return id, nil
}

func (q *Queue) worker(number int) {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return

		case task := <-q.tasks:
			q.execute(number, task)
		}
	}
}

func (q *Queue) execute(number int, task Task) {
	log.Printf("WORKER_%d START job=%d", number, task.ID)

	if err := q.store.Start(task.ID); err != nil {
		log.Println(err)
		return
	}

	writer := &logWriter{
		store: q.store,
		id:    task.ID,
	}

	err := q.engine.Run(q.ctx, task.Pipeline, writer)

	status := "succeeded"
	message := ""

	if err != nil {
		status = "failed"
		message = err.Error()
	}

	if err := q.store.Finish(task.ID, status, message); err != nil {
		log.Println(err)
	}

	log.Printf(
		"WORKER_%d END job=%d status=%s",
		number, task.ID, status,
	)
}

func (q *Queue) Wait() {
	q.wg.Wait()
}
