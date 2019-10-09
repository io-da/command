# [Go](https://golang.org/) Command Bus
A command bus to demand all the things.  

[![Build Status](https://travis-ci.org/io-da/command.svg?branch=master)](https://travis-ci.org/io-da/command)
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
5. [Benchmarks](#Benchmarks)
6. [Examples](#Examples)

## Introduction
This library is intended for anyone looking to trigger application commands in a decoupled architecture.  
The _Bus_ provides the option to use _workers_ ([goroutines](https://gobyexample.com/goroutines)) to attempt handling the commands in **non-blocking** manner.  
Clean and simple codebase. **No reflection, no closures.**

## Getting Started

### Commands
Commands are any type that implements the _Command_ interface. Ideally they should contain immutable data.  
```go
type Command interface {
    ID() []byte
}
```

### Handlers
Handlers are any type that implements the _Handler_ interface. Handlers must be instantiated and provided to the bus on initialization.    
```go
type Handler interface {
    Handle(cmd Command) error
}
```

### Error Handlers
Error handlers are any type that implements the _ErrorHandler_ interface. Error handlers are optional (but advised) and provided to the bus using the ```bus.ErrorHandlers``` function.  
```go
type ErrorHandler interface {
    Handle(evt Event, err error)
}
```
Any time an error occurs within the bus, it will be passed on to the error handlers. This strategy can be used for decoupled error handling.

### The Bus
_Bus_ is the _struct_ that will be used to trigger all the application's commands.  
The _Bus_ should be instantiated and initialized on application startup. The initialization is separated from the instantiation for dependency injection purposes.  
The application should instantiate the _Bus_ once and then use it's reference to trigger all the commands.  
**The order in which the handlers are provided to the _Bus_ is always respected. Additionally a command may have multiple handlers.**

#### Tweaking Performance
The number of workers for async commands can be adjusted.
```go
bus.WorkerPoolSize(10)
```
If used, this function **must** be called **before** the _Bus_ is initialized. And it specifies the number of [goroutines](https://gobyexample.com/goroutines) used to handle async commands.  
In some scenarios increasing the value can drastically improve performance.  
It defaults to the value returned by ```runtime.GOMAXPROCS(0)```.  
  
The buffer size of the async commands queue can also be adjusted.  
Depending on the use case, this value may greatly impact performance.
```go
bus.QueueBuffer(100)
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
// command.ErrorInvalidCommand
// command.ErrorCommandBusNotInitialized
// command.ErrorCommandBusIsShuttingDown

if err := bus.handle(&Command{}); err != nil {
    switch(err.(type)) {
        case command.ErrorInvalidCommand:
            // do something
        case command.CommandBusNotInitializedError:
            // do something
        case command.CommandBusIsShuttingDownError:
            // do something
        default:
            // do something
    }
}

```
## Benchmarks
All the benchmarks are performed against batches of 1 million commands.  
All the benchmarks contain some overhead due to the usage of _sync.WaitGroup_.  
The command handlers use ```time.Sleep(time.Nanosecond * 200)``` for simulation purposes.  

| Benchmark Type | Time |
| :--- | :---: |
| Sync Commands | 531 ns/op |
| Async Commands | 476 ns/op |

## Examples

#### Example Commands
A simple ```struct``` command.
```go
type Foo struct {
    bar string
}
func (*Foo) ID() []byte {
    return []byte("FOO-UUID")
}
```

A ```string``` command.
```go
type Bar string
func (Bar) ID() []byte {
    return []byte("BAR-UUID")
}
```

#### Example Handlers
A command handler that logs every command triggered.
```go
type LoggerHandler struct {
}

func (hdl *LoggerHandler) Handle(cmd Command) error {
    log.Printf("command %T emitted", cmd)
    return nil
}
```

A command handler that listens to multiple command types.
```go
type FooBarHandler struct {
}

func (hdl *FooBarHandler) Handle(cmd Command) error {
    // a convenient way to assert multiple command types.
    switch cmd := cmd.(type) {
    case *Foo, Bar:
        // handler logic
    }
    return nil
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
    
    // initialize the bus with all of the application's command handlers
    bus.Initialize(
        // this handler will always be executed first
        &LoggerHandler{},
        // this one second
        &FooBarHandler{},
    )
    
    // trigger commands!
    bus.Handle(&Foo{})
    bus.Handle(Bar("bar"))
}
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)