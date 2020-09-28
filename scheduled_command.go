package command

import "github.com/io-da/schedule"

type scheduledCommand struct {
	cmd Command
	sch *schedule.Schedule
}

func newScheduledCommand(cmd Command, sch *schedule.Schedule) *scheduledCommand {
	return &scheduledCommand{
		cmd: cmd,
		sch: sch,
	}
}
