package apiservice

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/wurt83ow/gophstream/internal/models"
	"github.com/wurt83ow/gophstream/internal/workerpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type External interface {
	SendMessageToKafka(message models.Message) (int, error)
}

type Log interface {
	Info(string, ...zapcore.Field)
}

type Storage interface {
	GetMessages(context.Context, models.Filter, models.Pagination) ([]models.Message, error)
	UpdateMessagesProcessed(context.Context, []int) error
}

type Pool interface {
	// NewTask(f func(interface{}) error, data interface{}) *workerpool.Task
	AddTask(task *workerpool.Task)
}

type ApiService struct {
	ctx          context.Context
	results      chan interface{}
	wg           sync.WaitGroup
	cancelFunc   context.CancelFunc
	external     External
	pool         Pool
	storage      Storage
	log          Log
	taskInterval int
}

func NewApiService(ctx context.Context, external External, pool Pool, storage Storage,
	log Log, taskInterval func() string,
) *ApiService {
	taskInt, err := strconv.Atoi(taskInterval())
	if err != nil {
		log.Info("cannot convert concurrency option: ", zap.Error(err))

		taskInt = 3000
	}

	return &ApiService{
		ctx:          ctx,
		results:      make(chan interface{}),
		wg:           sync.WaitGroup{},
		cancelFunc:   nil,
		external:     external,
		pool:         pool,
		storage:      storage,
		log:          log,
		taskInterval: taskInt,
	}
}

func (a *ApiService) Start() {
	a.ctx, a.cancelFunc = context.WithCancel(a.ctx)
	a.wg.Add(1)
	go a.ProcessMessages(a.ctx)
}

func (a *ApiService) Stop() {
	a.cancelFunc()
	a.wg.Wait()
}

func (a *ApiService) ProcessMessages(ctx context.Context) {
	t := time.NewTicker(time.Duration(a.taskInterval) * time.Millisecond)

	result := make([]int, 0)

	var dmx sync.RWMutex

	dmx.RLock()
	defer dmx.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return
		case job := <-a.results:
			j, ok := job.(int)
			if ok {
				result = append(result, j)
			}
		case <-t.C:

			processed := false
			filter := models.Filter{
				Processed: &processed,
			}

			pagination := models.Pagination{
				Limit: 100,
			}

			messages, err := a.storage.GetMessages(ctx, filter, pagination)

			if err != nil {

				return
			}

			a.CreateTask(messages)

			if len(result) != 0 {
				a.doWork(result)
				result = nil
			}
		}
	}
}

// AddResults adds result to pool.
func (a *ApiService) AddResults(result interface{}) {
	a.results <- result
}

func (a *ApiService) GetResults() <-chan interface{} {
	// close(p.results)
	return a.results
}

func (a *ApiService) CreateTask(messages []models.Message) {
	var task *workerpool.Task

	for _, message := range messages {

		task = workerpool.NewTask(func(data interface{}) error {

			msg, ok := data.(models.Message)
			if ok { // type assertion failed
				msg_id, err := a.external.SendMessageToKafka(msg)

				if err != nil {
					return fmt.Errorf("failed to create order task: %w", err)
				}
				a.log.Info("processed task: ", zap.String("usefinfo", fmt.Sprintf("%d%d", msg.ID)))
				a.AddResults(msg_id)
			}

			return nil
		}, message)
		a.pool.AddTask(task)
	}
}

func (a *ApiService) doWork(ids []int) {
	// perform a group update of the massages table (field Processed)
	err := a.storage.UpdateMessagesProcessed(a.ctx, ids)
	if err != nil {
		a.log.Info("errors when updating order status: ", zap.Error(err))
	}

}
