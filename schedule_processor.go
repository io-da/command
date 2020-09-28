package command

import (
	"github.com/google/uuid"
	"sync"
	"sync/atomic"
	"time"
)

type scheduleProcessor struct {
	sync.Mutex
	bus               *Bus
	scheduledCommands map[uuid.UUID]*scheduledCommand
	triggerSignal     chan bool
	shuttingDown      *uint32
	sleepTimer        *time.Timer
	sleepUntil        time.Time
}

func newScheduleProcessor(bus *Bus) *scheduleProcessor {
	pro := &scheduleProcessor{
		bus:               bus,
		scheduledCommands: make(map[uuid.UUID]*scheduledCommand),
		triggerSignal:     make(chan bool, 1),
		shuttingDown:      new(uint32),
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
	atomic.CompareAndSwapUint32(pro.shuttingDown, 0, 1)
	pro.trigger()
}

func (pro *scheduleProcessor) process() {
	for atomic.LoadUint32(pro.shuttingDown) == 0 {
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
				_ = pro.bus.HandleAsync(schCmd.cmd)
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
	pro.triggerSignal <- true
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

	return pro.sleepUntil.Sub(time.Now())
}

func (pro *scheduleProcessor) updateSleepTimer(d time.Duration) {
	if pro.sleepTimer == nil {
		pro.sleepTimer = time.NewTimer(d)
		return
	}
	pro.sleepTimer.Reset(d)
}
