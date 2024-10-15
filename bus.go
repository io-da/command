package command

import (
	"runtime"

	"github.com/google/uuid"
	"github.com/io-da/schedule"
)

// Bus is the only struct exported and required for the command bus usage.
// The Bus should be instantiated using the NewBus function.
type Bus struct {
	workerPoolSize     int
	queueBuffer        int
	initialized        *flag
	shuttingDown       *flag
	workers            *counter
	handlers           map[Identifier]Handler
	errorHandlers      []ErrorHandler
	middlewares        []Middleware
	asyncCommandsQueue chan *Async
	closed             chan bool
	scheduleProcessor  *scheduleProcessor
}

// NewBus instantiates the Bus struct.
// The Initialization of the Bus is performed separately (Initialize function) for dependency injection purposes.
func NewBus() *Bus {
	bus := &Bus{
		workerPoolSize: runtime.GOMAXPROCS(0),
		queueBuffer:    100,
		initialized:    newFlag(),
		shuttingDown:   newFlag(),
		workers:        newCounter(),
		handlers:       make(map[Identifier]Handler),
		errorHandlers:  make([]ErrorHandler, 0),
		middlewares:    make([]Middleware, 0),
		closed:         make(chan bool),
	}
	bus.scheduleProcessor = newScheduleProcessor(bus)
	return bus
}

// SetWorkerPoolSize may optionally be used to tweak the worker pool size for async commands.
// It can only be adjusted *before* the bus is initialized.
// It defaults to the value returned by runtime.GOMAXPROCS(0).
func (bus *Bus) SetWorkerPoolSize(workerPoolSize int) {
	if !bus.initialized.enabled() {
		bus.workerPoolSize = workerPoolSize
	}
}

// SetQueueBuffer may optionally be used to tweak the buffer size of the async commands queue.
// This value may have high impact on performance depending on the use case.
// It can only be adjusted *before* the bus is initialized.
// It defaults to 100.
func (bus *Bus) SetQueueBuffer(queueBuffer int) {
	if !bus.initialized.enabled() {
		bus.queueBuffer = queueBuffer
	}
}

// SetErrorHandlers may optionally be used to provide a list of error handlers.
// They will receive any error thrown during the command process.
// Error handlers may only be provided *before* the bus is initialized.
func (bus *Bus) SetErrorHandlers(hdls ...ErrorHandler) {
	if !bus.initialized.enabled() {
		bus.errorHandlers = hdls
	}
}

// SetMiddlewares may optionally be used to provide a list of middlewares.
// They will receive and process every command that is about to be handled.
// Middlewares may be used to modify or completely prevent the command execution.
// Middlewares are responsible for the execution of the next step:
//
//	Handle(cmd Command, next Next) (any, error) {
//	 return next(cmd)
//	}
//
// *The order in which the middlewares are provided to the Bus is always respected*.
// Middlewares may only be provided *before* the bus is initialized.
func (bus *Bus) SetMiddlewares(mdls ...Middleware) {
	if !bus.initialized.enabled() {
		bus.middlewares = mdls
	}
}

// Initialize the command bus by providing the list of handlers.
// There can only be one handler per command.
func (bus *Bus) Initialize(hdls ...Handler) error {
	if bus.initialized.enable() {
		closureHandlerProvided := false
		for _, hdl := range hdls {
			if _, exists := bus.handlers[hdl.Handles()]; exists {
				return OneHandlerPerCommandError
			}
			bus.handlers[hdl.Handles()] = hdl
			if hdl.Handles() == ClosureIdentifier {
				closureHandlerProvided = true
			}
		}
		if !closureHandlerProvided {
			hdl := &ClosureHandler{}
			bus.handlers[hdl.Handles()] = hdl
		}
		bus.asyncCommandsQueue = make(chan *Async, bus.queueBuffer)
		for i := 0; i < bus.workerPoolSize; i++ {
			bus.workers.increment()
			go bus.worker(bus.asyncCommandsQueue, bus.closed)
		}
	}
	return nil
}

// Handle processes the command synchronously through their respective handler.
func (bus *Bus) Handle(cmd Command) (any, error) {
	hdl, err := bus.getHandler(cmd)
	if err != nil {
		return nil, err
	}
	return bus.handle(hdl, cmd)
}

// HandleAsync processes the command asynchronously using workers through their respective handler.
// It also returns an *Async struct which allows clients to optionally ```Await``` for the command to be processed.
func (bus *Bus) HandleAsync(cmd Command) (*Async, error) {
	async, err := bus.prepareAsync(cmd)
	if err != nil {
		return nil, err
	}
	bus.asyncCommandsQueue <- async
	return async, nil
}

// HandleAsyncList processes the provided commands asynchronously using workers through their respective handler.
// It also returns an *AsyncList struct which allows clients to optionally ```Await``` for the commands respectively.
func (bus *Bus) HandleAsyncList(cmds ...Command) (*AsyncList, error) {
	asl := &AsyncList{make([]*Async, len(cmds))}
	for i, cmd := range cmds {
		async, err := bus.prepareAsync(cmd)
		if err != nil {
			return nil, err
		}
		asl.cmds[i] = async
	}
	for _, async := range asl.cmds {
		bus.asyncCommandsQueue <- async
	}
	return asl, nil
}

// Schedule allows commands to be scheduled to be executed asynchronously.
// Check https://github.com/io-da/schedule for ```*Schedule``` usage.
func (bus *Bus) Schedule(cmd Command, sch *schedule.Schedule) (*uuid.UUID, error) {
	hdl, err := bus.getHandler(cmd)
	if err != nil {
		return nil, err
	}
	key := bus.scheduleProcessor.add(newScheduledCommand(hdl, cmd, sch))
	return &key, nil
}

// RemoveScheduled removes previously scheduled commands.
func (bus *Bus) RemoveScheduled(keys ...uuid.UUID) {
	bus.scheduleProcessor.remove(keys...)
}

// Shutdown the command bus gracefully.
// *Async commands accessed while shutting down will be disregarded*.
func (bus *Bus) Shutdown() {
	if bus.shuttingDown.enable() {
		go bus.shutdown()
	}
}

//-----Internal------//

func (bus *Bus) worker(asyncCommandsQueue <-chan *Async, closed chan<- bool) {
	for async := range asyncCommandsQueue {
		if async == nil {
			break
		}
		bus.handleAsync(async)
	}
	closed <- true
}

func (bus *Bus) prepareAsync(cmd Command) (*Async, error) {
	hdl, err := bus.getHandler(cmd)
	if err != nil {
		return nil, err
	}
	return newAsync(hdl, cmd), nil
}

func (bus *Bus) handleAsync(async *Async) {
	data, err := bus.handle(async.hdl, async.cmd)
	if err != nil {
		async.fail(err)
		return
	}
	async.success(data)
}

func (bus *Bus) handle(hdl Handler, cmd Command) (data any, err error) {
	data, err = bus.handleMiddlewares(hdl, cmd, 0)
	if err != nil {
		data = nil
		bus.error(cmd, err)
	}
	return
}

func (bus *Bus) handleMiddlewares(hdl Handler, cmd Command, currentMdlIdx int) (data any, err error) {
	if len(bus.middlewares) == 0 {
		return hdl.Handle(cmd)
	}
	mdl := bus.middlewares[currentMdlIdx]
	currentMdlIdx++
	if currentMdlIdx < len(bus.middlewares) {
		return mdl.Handle(cmd, func(cmd Command) (any, error) {
			return bus.handleMiddlewares(hdl, cmd, currentMdlIdx)
		})
	}
	return mdl.Handle(cmd, hdl.Handle)
}

func (bus *Bus) shutdown() {
	for !bus.workers.is(0) {
		bus.asyncCommandsQueue <- nil
		<-bus.closed
		bus.workers.decrement()
	}
	bus.scheduleProcessor.shutdown()
	bus.initialized.disable()
	bus.shuttingDown.disable()
}

func (bus *Bus) getHandler(cmd Command) (hdl Handler, err error) {
	if cmd == nil {
		err = InvalidCommandError
		bus.error(cmd, err)
		return
	}
	if !bus.initialized.enabled() {
		err = BusNotInitializedError
		bus.error(cmd, err)
		return
	}
	if bus.shuttingDown.enabled() {
		err = BusIsShuttingDownError
		bus.error(cmd, err)
		return
	}
	hdl, ok := bus.handlers[cmd.Identifier()]
	if !ok {
		err = HandlerNotFoundError
		bus.error(cmd, err)
	}
	return
}

func (bus *Bus) error(cmd Command, err error) {
	for _, errHdl := range bus.errorHandlers {
		errHdl.Handle(cmd, err)
	}
}
