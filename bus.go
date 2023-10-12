package command

import (
	"github.com/google/uuid"
	"github.com/io-da/schedule"
	"runtime"
	"sync/atomic"
)

// Bus is the only struct exported and required for the command bus usage.
// The Bus should be instantiated using the NewBus function.
type Bus struct {
	workerPoolSize     int
	queueBuffer        int
	initialized        *uint32
	shuttingDown       *uint32
	workers            *uint32
	handlers           map[Identifier]Handler
	errorHandlers      []ErrorHandler
	asyncCommandsQueue chan *process[*ResultAsync]
	closed             chan bool
	scheduleProcessor  *scheduleProcessor
}

// NewBus instantiates the Bus struct.
// The Initialization of the Bus is performed separately (Initialize function) for dependency injection purposes.
func NewBus() *Bus {
	bus := &Bus{
		workerPoolSize: runtime.GOMAXPROCS(0),
		queueBuffer:    100,
		initialized:    new(uint32),
		shuttingDown:   new(uint32),
		workers:        new(uint32),
		handlers:       make(map[Identifier]Handler),
		errorHandlers:  make([]ErrorHandler, 0),
		closed:         make(chan bool),
	}
	bus.scheduleProcessor = newScheduleProcessor(bus)
	return bus
}

// SetWorkerPoolSize may optionally be used to tweak the worker pool size for async commands.
// It can only be adjusted *before* the bus is initialized.
// It defaults to the value returned by runtime.GOMAXPROCS(0).
func (bus *Bus) SetWorkerPoolSize(workerPoolSize int) {
	if !bus.isInitialized() {
		bus.workerPoolSize = workerPoolSize
	}
}

// SetQueueBuffer may optionally be used to tweak the buffer size of the async commands queue.
// This value may have high impact on performance depending on the use case.
// It can only be adjusted *before* the bus is initialized.
// It defaults to 100.
func (bus *Bus) SetQueueBuffer(queueBuffer int) {
	if !bus.isInitialized() {
		bus.queueBuffer = queueBuffer
	}
}

// SetErrorHandlers may optionally be used to provide a list of error handlers.
// They will receive any error thrown during the command process.
func (bus *Bus) SetErrorHandlers(hdls ...ErrorHandler) {
	if !bus.isInitialized() {
		bus.errorHandlers = hdls
	}
}

// Initialize the command bus.
func (bus *Bus) Initialize(hdls ...Handler) error {
	if bus.initialize() {
		for _, hdl := range hdls {
			if _, exists := bus.handlers[hdl.Handles()]; exists {
				return OneHandlerPerCommandError
			}
			bus.handlers[hdl.Handles()] = hdl
		}
		bus.asyncCommandsQueue = make(chan *process[*ResultAsync], bus.queueBuffer)
		for i := 0; i < bus.workerPoolSize; i++ {
			bus.workerUp()
			go bus.worker(bus.asyncCommandsQueue, bus.closed)
		}
	}
	return nil
}

// HandleAsync the command using the workers asynchronously.
func (bus *Bus) HandleAsync(cmd Command) (*ResultAsync, error) {
	if err := bus.isValid(cmd); err != nil {
		return nil, err
	}
	prs := newProcess(cmd, newResultAsync())
	bus.asyncCommandsQueue <- prs
	return prs.res, nil
}

// Handle the command synchronously.
func (bus *Bus) Handle(cmd Command) (*Result, error) {
	if err := bus.isValid(cmd); err != nil {
		return nil, err
	}
	prs := newProcess(cmd, newResult())
	if err := bus.handle(prs.cmd, prs.res); err != nil {
		return nil, err
	}
	return prs.res, nil
}

// Schedule allows commands to be scheduled to be executed asynchronously.
func (bus *Bus) Schedule(cmd Command, sch *schedule.Schedule) (*uuid.UUID, error) {
	if err := bus.isValid(cmd); err != nil {
		return nil, err
	}
	key := bus.scheduleProcessor.add(newScheduledCommand(cmd, sch))
	return &key, nil
}

// RemoveScheduled removes previously scheduled commands.
func (bus *Bus) RemoveScheduled(keys ...uuid.UUID) {
	bus.scheduleProcessor.remove(keys...)
}

// Shutdown the command bus gracefully.
// *Async commands handled while shutting down will be disregarded*.
func (bus *Bus) Shutdown() {
	if atomic.CompareAndSwapUint32(bus.shuttingDown, 0, 1) {
		go bus.shutdown()
	}
}

//-----Private Functions------//

func (bus *Bus) initialize() bool {
	return atomic.CompareAndSwapUint32(bus.initialized, 0, 1)
}

func (bus *Bus) isInitialized() bool {
	return atomic.LoadUint32(bus.initialized) == 1
}

func (bus *Bus) isShuttingDown() bool {
	return atomic.LoadUint32(bus.shuttingDown) == 1
}

func (bus *Bus) worker(asyncCommandsQueue <-chan *process[*ResultAsync], closed chan<- bool) {
	for prs := range asyncCommandsQueue {
		if prs == nil {
			break
		}
		bus.handleAsync(prs.cmd, prs.res)
	}
	closed <- true
}

func (bus *Bus) handle(cmd Command, res *Result) error {
	if hdl, ok := bus.handlers[cmd.Identifier()]; ok {
		data, err := hdl.Handle(cmd)
		if err != nil {
			bus.error(cmd, err)
			return err
		}
		if data != nil {
			res.setReturn(data)
		}
	}
	return nil
}

func (bus *Bus) handleAsync(cmd Command, res *ResultAsync) {
	if err := bus.handle(cmd, res.Result); err != nil {
		res.fail(err)
		return
	}
	res.done()
}

func (bus *Bus) workerUp() {
	atomic.AddUint32(bus.workers, 1)
}

func (bus *Bus) workerDown() {
	atomic.AddUint32(bus.workers, ^uint32(0))
}

func (bus *Bus) shutdown() {
	for atomic.LoadUint32(bus.workers) > 0 {
		bus.asyncCommandsQueue <- nil
		<-bus.closed
		bus.workerDown()
	}
	bus.scheduleProcessor.shutdown()
	atomic.CompareAndSwapUint32(bus.initialized, 1, 0)
	atomic.CompareAndSwapUint32(bus.shuttingDown, 1, 0)
}

func (bus *Bus) isValid(cmd Command) (err error) {
	if cmd == nil {
		err = InvalidCommandError
		bus.error(cmd, err)
		return
	}
	if !bus.isInitialized() {
		err = BusNotInitializedError
		bus.error(cmd, err)
		return
	}
	if bus.isShuttingDown() {
		err = BusIsShuttingDownError
		bus.error(cmd, err)
		return
	}
	if _, ok := bus.handlers[cmd.Identifier()]; !ok {
		err = HandlerNotFoundError
		bus.error(cmd, err)
		return
	}
	return
}

func (bus *Bus) error(cmd Command, err error) {
	for _, errHdl := range bus.errorHandlers {
		errHdl.Handle(cmd, err)
	}
}
