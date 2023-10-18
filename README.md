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
4. [Middlewares](#Middlewares)
5. [The Bus](#The-Bus)  
   1. [Handling Commands](#Handling-Commands)  
   2. [Tweaking Performance](#Tweaking-Performance)  
   3. [Shutting Down](#Shutting-Down)  
   4. [Available Errors](#Available-Errors)
   5. [Scheduled Commands](#Scheduled-Commands)
6. [Benchmarks](#Benchmarks)
7. [Examples](#Examples)

## Introduction
This library is intended for anyone looking to trigger application commands in a decoupled architecture.  
The _Bus_ provides the option to use _workers_ ([goroutines](https://gobyexample.com/goroutines)) to attempt handling the commands in **non-blocking** manner.  
Clean and simple codebase.

## Getting Started

### Commands
Commands are any type that implements the _Command_ interface. Ideally they should contain immutable data.  
It is also possible to provide a closure to the bus.
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


### Middlewares
Middlewares are any type that implements the _InwardMiddleware_ or the _OutwardMiddleware_ interface.  
Middlewares are optional and provided to the _Bus_ using the ```bus.SetInwardMiddlewares``` and ```bus.SetOutwardMiddlewares``` functions.  
An _InwardMiddleware_ handles every command before it is provided to its respective handler.  
An _OutwardMiddleware_ handles every command that was successfully processed by its respective handler. These middlewares are also provided with the data or error returned by the command handler. Allowing potential data/error handling, such as transformations.

**The order in which the middlewares are provided to the Bus is always respected. Additionally, if a middleware returns an error, it interrupts the flow and the command is no longer passed along to the next step.**
```go
type InwardMiddleware interface {
    HandleInward(cmd Command) error
}

type OutwardMiddleware interface {
    HandleOutward(cmd Command, data any, err error) error
}
```

### The Bus
_Bus_ is the _struct_ that will be used to trigger all the application's commands.  
The _Bus_ should be instantiated and initialized on application startup. The initialization is separated from the instantiation for dependency injection purposes.  
The application should instantiate the _Bus_ once and then use it's reference to trigger all the commands.  
**There can only be one _Handler_ per _Command_**.

#### Handling Commands
The _Bus_ provides multiple ways to handle commands.  
> **Synchronous**. The bus processes the command immediately and returns the result from the handler.
>```go
>data, err := bus.Handle(&FooBar{})
>```

> **Asynchronous**. The bus processes the command using workers. It is no-blocking.  
> It is possible however to _Await_ for the command to finish being processed.
>```go
>as, _ := bus.HandleAsync(&FooBar{})
>// do something
>data, err := as.Await()
>```

> **Closures**. The bus also accepts a closure to be provided.  
> It will be handled in an asynchronous manner using workers. These also support _Await_.
>```go
>as, _ := bus.HandleClosure(func() (data any, err error) {
>   return "foo bar", nil
>})
>// do something
>data, err := as.Await()
>```

> **Schedule**. The bus will use a schedule processor to handle the provided command according to a _*Schedule_  struct.  
> More information about _*Schedule_ can be found [here](https://github.com/io-da/schedule).
>```go
>uuid, err := bus.Schedule(&FooBar{}, schedule.At(time.Now())))
>// if the scheduled command needs to be removed during runtime.
>bus.RemoveScheduled(uuid)
>```

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
Below is a list of errors that can occur when calling ```bus.Initialize```, ```bus.Handle```, ```bus.HandleAsync``` and ```bus.Schedule```.  
```go
command.InvalidCommandError
command.BusNotInitializedError
command.BusIsShuttingDownError
command.OneHandlerPerCommandError
command.HandlerNotFoundError
```

#### Scheduled Commands
Since ```1.2```, the bus also has built in support for [github.com/io-da/schedule](https://github.com/io-da/schedule).  
Using ```bus.Schedule```, one may schedule a command to be processed at certain times or even following a cron like pattern.

## Benchmarks
All the benchmarks are performed with command handlers calculating the fibonacci of 1000.  
CPU: 11th Gen Intel(R) Core(TM) i9-11950H @ 2.60GHz  

| Benchmark Type | Time |
| :--- | :---: |
| Sync Commands | 15828 ns/op |
| Async Commands | 2808 ns/op |

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