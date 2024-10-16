package command

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type scheduleProcessor struct {
	sync.Mutex
	bus               *Bus
	scheduledCommands map[uuid.UUID]*scheduledCommand
	triggerSignal     chan bool
	shuttingDown      *flag
	sleepTimer        *time.Timer
	sleepUntil        time.Time
}

func newScheduleProcessor(bus *Bus) *scheduleProcessor {
	pro := &scheduleProcessor{
		bus:               bus,
		scheduledCommands: make(map[uuid.UUID]*scheduledCommand),
		triggerSignal:     make(chan bool, 1),
		shuttingDown:      newFlag(),
	}
	go pro.process()
	return pro
}

func (pro *scheduleProcessor) add(schCmd *scheduledCommand) uuid.UUID {
	pro.Lock()
	key := uuid.New()
	pro.scheduledCommands[key] = schCmd
	pro.Unlock()
	pro.trigger()
	return key
}

func (pro *scheduleProcessor) remove(keys ...uuid.UUID) {
	pro.Lock()
	for _, key := range keys {
		delete(pro.scheduledCommands, key)
	}
	pro.Unlock()
	pro.trigger()
}

func (pro *scheduleProcessor) shutdown() {
	if pro.shuttingDown.enable() {
		pro.trigger()
	}
}

func (pro *scheduleProcessor) process() {
	for !pro.shuttingDown.enabled() {
		pro.Lock()
		now := time.Now()
		pro.sleepUntil = time.Time{}
		for key, schCmd := range pro.scheduledCommands {
			following := schCmd.sch.Following()
			if following.IsZero() {
				_ = schCmd.sch.Next()
				following = schCmd.sch.Following()
			}

			if now.After(following) || now.Equal(following) {
				async := newAsync(schCmd.hdl, schCmd.cmd)
				pro.bus.asyncCommandsQueue <- async
				if err := schCmd.sch.Next(); err != nil {
					delete(pro.scheduledCommands, key)
					continue
				}
				following = schCmd.sch.Following()
			}
			pro.updateSleepUntil(following)
		}
		pro.updateSleepTimer(pro.determineSleepDuration())
		pro.Unlock()

		// allow the processor to be triggered either with timer or directly
		select {
		case <-pro.sleepTimer.C:
		case <-pro.triggerSignal:
		}
	}
}

func (pro *scheduleProcessor) trigger() {
	select {
	case pro.triggerSignal <- true:
	default:
	}
}

func (pro *scheduleProcessor) updateSleepUntil(nextTrigger time.Time) {
	if pro.sleepUntil.IsZero() || nextTrigger.Before(pro.sleepUntil) {
		pro.sleepUntil = nextTrigger
	}
}

func (pro *scheduleProcessor) determineSleepDuration() time.Duration {
	if pro.sleepUntil.IsZero() || len(pro.scheduledCommands) <= 0 {
		return time.Hour
	}
	return time.Until(pro.sleepUntil)
}

func (pro *scheduleProcessor) updateSleepTimer(d time.Duration) {
	if pro.sleepTimer == nil {
		pro.sleepTimer = time.NewTimer(d)
		return
	}
	pro.sleepTimer.Reset(d)
}
