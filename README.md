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
Middlewares are any type that implements the _Middleware_ interface.  
```go
type Middleware interface {
    Handle(cmd Command, next Next) (any, error)
}
```
Middlewares are optional and provided to the _Bus_ using the ```bus.SetMiddlewares``` function.  
A _Middleware_ handles every command before it is provided to its respective handler.  
Middlewares may be used to modify or completely prevent the command execution.
Middlewares are responsible for the execution of the next step:
```go
func (Middleware) Handle(cmd Command, next Next) (any, error) {
    // do something before...
    data, err := next(cmd)
    // do something after...
    return data, err
}
```
**The order in which the middlewares are provided to the Bus is always respected.**

### The Bus
_Bus_ is the _struct_ that will be used to trigger all the application's commands.  
The _Bus_ should be instantiated and initialized on application startup. The initialization is separated from the instantiation for dependency injection purposes.  
The application should instantiate the _Bus_ once and then use it's reference to trigger all the commands.  
**There can only be one _Handler_ per _Command_**.

#### Handling Commands
The _Bus_ provides multiple ways to handle commands.  
##### Synchronous
> The bus processes the command immediately and returns the result from the handler.
>```go
>data, err := bus.Handle(&FooBar{})
>```

##### Asynchronous
> The bus processes the command using workers. It is no-blocking.  
> It is possible however to _Await_ for the command to finish being processed.
>```go
>as, _ := bus.HandleAsync(&FooBar{})
>// do something
>data, err := as.Await()
>```

##### Asynchronous List
> The bus processes the provided commands using workers. It is no-blocking.  
> It is possible however to _Await_ for these commands to finish being processed.
> The return type is a _*AsyncList_. This struct exposes a few methods:
> - _Await_ waits for all the commands to finish processing similarly to a regular _*Async_ type. However, it returns `[]any, error`. The order of the returned results matches the order of the provided commands. The error joins all the resulting errors.
> - _AwaitIterator_ returns an iterator to process the results in the order they are handled. This means that if multiple commands are issued, the fastest ones will be received first. The value that is returned by each iteration is a _AsyncResult_. Aside from the _Data_ and _Error_, this struct also exposes the _Index_ matching the order of the issued commands.
>```go
>asl, _ := bus.HandleAsyncList(&FooBar{}, &FooBar2{}, &FooBar3{})
>// do something
>data, err := asl.Await()
>```

##### Closures
> The bus also accepts closure commands to be provided.  
> These will be handled similarly to any other command. Using the type _Closure_.
>```go
>as, _ := bus.HandleAsync(Closure(func() (data any, err error) {
>   return "foo bar", nil
>}))
>// do something
>data, err := as.Await()
>```

##### Schedule
> The bus will use a schedule processor to handle the provided command according to a _*Schedule_  struct.  
> More information about _*Schedule_ can be found [here](https://github.com/io-da/schedule).
>```go
>uuid, err := bus.Schedule(&FooBar{}, schedule.At(time.Now()))
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
command.EmptyAwaitListError
command.InvalidClosureCommandError
```

#### Scheduled Commands
Since ```1.2```, the bus also has built in support for [github.com/io-da/schedule](https://github.com/io-da/schedule).  
Using ```bus.Schedule```, one may schedule a command to be processed at certain times or even following a cron like pattern.

## Benchmarks
All the benchmarks are performed with command handlers calculating the fibonacci of 100.
CPU: Apple M3 Pro

| Benchmark Type | Time |
| :--- | :---: |
| Sync Commands | 660 ns/op |
| Async Commands | 425 ns/op |

For reference, here is the benchmark of the function used (fibonacci of 100) for simulation purposes, when run directly:  
| Benchmark Type | Time |
| :--- | :---: |
| fibonacci(100) | 643 ns/op |

These results imply that the infrastructure induces a small overhead of about `10-20ns`.  
Naturally, the reason for the async's method improved performance is the fact that the executions of the fibonacci function are being spread over multiple workers.

## Examples
An optional constants list of _Command_ identifiers (idiomatic ```enum```) for consistency
```go
const (
	FooCommand Identifier = "FooCommand"
	BarCommand Identifier = "BarCommand"
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

## Change Log
The change log is generated using the tool [git-chglog](https://github.com/git-chglog/).  
Thanks to their amazing work, the CHANGELOG.md can be easily regenerated using the following command:
```sh
git-chglog -o CHANGELOG.md
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)