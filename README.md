# [Go](https://golang.org/) Command Bus
A command bus to demand all the things.  

[![Maintainability](https://api.codeclimate.com/v1/badges/320d3b5a036178276900/maintainability)](https://codeclimate.com/github/io-da/command/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/320d3b5a036178276900/test_coverage)](https://codeclimate.com/github/io-da/command/test_coverage)
[![GoDoc](https://godoc.org/github.com/io-da/command?status.svg)](https://godoc.org/github.com/io-da/command)

## Installation
``` go get github.com/io-da/command ```

## Overview
1. [Commands](#Commands)
2. [Handlers](#Handlers)
3. [Error Handlers](#Error-Handlers)
4. [The Bus](#The-Bus)  
   1. [Tweaking Performance](#Tweaking-Performance)  
   2. [Shutting Down](#Shutting-Down)  
   3. [Available Errors](#Available-Errors)
   3. [Scheduled Commands](#Scheduled-Commands)
5. [Benchmarks](#Benchmarks)
6. [Examples](#Examples)

## Introduction
This library is intended for anyone looking to trigger application commands in a decoupled architecture.  
The _Bus_ provides the option to use _workers_ ([goroutines](https://gobyexample.com/goroutines)) to attempt handling the commands in **non-blocking** manner.  
Clean and simple codebase.

## Getting Started

### Commands
Commands are any type that implements the _Command_ interface. Ideally they should contain immutable data.  
```go
type Identifier int64

type Command interface {
    Identifier() Identifier
}
```

### Handlers
Handlers are any type that implements the _Handler_ interface. Handlers must be instantiated and provided to the _Bus_ on initialization.  
The _Bus_ is initialized using the function ```bus.Initialize```. The _Bus_ will then use the _Identifier_ of the handlers to know which _Command_ to process.
```go
type Handler interface {
    Handle(cmd Command) (any, error)
    Handles() Identifier
}
```

### Error Handlers
Error handlers are any type that implements the _ErrorHandler_ interface. Error handlers are optional (but advised) and provided to the _Bus_ using the ```bus.SetErrorHandlers``` function.  
```go
type ErrorHandler interface {
    Handle(cmd Command, err error)
}
```
Any time an error occurs within the bus, it will be passed on to the error handlers. This strategy can be used for decoupled error handling.

### The Bus
_Bus_ is the _struct_ that will be used to trigger all the application's commands.  
The _Bus_ should be instantiated and initialized on application startup. The initialization is separated from the instantiation for dependency injection purposes.  
The application should instantiate the _Bus_ once and then use it's reference to trigger all the commands.  
**There can only be one _Handler_ per _Command_**.

#### Tweaking Performance
The number of workers for async commands can be adjusted.
```go
bus.SetWorkerPoolSize(10)
```
If used, this function **must** be called **before** the _Bus_ is initialized. And it specifies the number of [goroutines](https://gobyexample.com/goroutines) used to handle async commands.  
In some scenarios increasing the value can drastically improve performance.  
It defaults to the value returned by ```runtime.GOMAXPROCS(0)```.  
  
The buffer size of the async commands queue can also be adjusted.  
Depending on the use case, this value may greatly impact performance.
```go
bus.SetQueueBuffer(100)
```
If used, this function **must** be called **before** the _Bus_ is initialized.  
It defaults to 100.  

#### Shutting Down
The _Bus_ also provides a shutdown function that attempts to gracefully stop the command bus and all its routines.
```go
bus.Shutdown()
```  
**This function will block until the bus is fully stopped.**

#### Available Errors
Below is a list of errors that can occur when calling ```bus.HandleAsync or bus.Handle```.  
```go
command.InvalidCommandError
command.BusNotInitializedError
command.BusIsShuttingDownError
command.HandlerNotFoundError
```

#### Scheduled Commands
Since ```1.2```, the bus also has built in support for [github.com/io-da/schedule](https://github.com/io-da/schedule).  
Using ```bus.Schedule```, one may schedule a command to be processed at certain times or even following a cron like pattern.

## Benchmarks
All the benchmarks are performed against batches of 1 million commands.  
All the benchmarks contain some overhead due to the usage of _sync.WaitGroup_.  
The command handlers use ```time.Sleep(time.Nanosecond * 200)``` for simulation purposes.  

| Benchmark Type | Time |
| :--- | :---: |
| Sync Commands | 531 ns/op |
| Async Commands | 476 ns/op |

## Examples
An optional constants list of _Command_ identifiers (idiomatic ```enum```) for consistency
```go
const (
   Unidentified Identifier = iota
   FooCommand
   BarCommand
)
```
#### Example Commands
A simple ```struct``` command.
```go

type fooCommand struct {
    bar string
}
func (*fooCommand) Identifier() Identifier {
    return FooCommand
}
```

A ```string``` command.
```go
type barCommand string
func (barCommand) Identifier() Identifier {
    return BarCommand
}
```

#### Example Handlers
A couple of empty respective handlers.
```go

type fooHandler struct{}

func (hdl *fooHandler) Handles() Identifier {
    return FooCommand
}

func (hdl *fooHandler) Handle(cmd Command) (data any, err error) {
    // handle FooCommand
    return
}

type barHandler struct{}

func (hdl *barHandler) Handles() Identifier {
    return BarCommand
}

func (hdl *barHandler) Handle(cmd Command) (data any, err error) {
    // handle BarCommand
    return
}
```

#### Putting it together
Initialization and usage of the exemplified commands and handlers
```go
import (
    "github.com/io-da/command"
)

func main() {
    // instantiate the bus (returns *command.Bus)
    bus := command.NewBus()
    
    // initialize the bus with all the application's command handlers
    bus.Initialize(
        &fooHandler{},
        &barHandler{},
    )
    
    // trigger commands!
    // sync
    bus.Handle(&fooCommand{})
    // async
    bus.HandleAsync(barCommand{})
    // async await
    res, _ := bus.HandleAsync(&fooCommand{}) 
    // do something
    res.Await()
    // scheduled to run every day
    sch := schedule.As(schedule.Cron().EveryDay())
    bus.Schedule(&fooCommand{}, sch)
}
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)