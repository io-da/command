package command

import "github.com/io-da/schedule"

type scheduledCommand struct {
	hdl Handler
	cmd Command
	sch *schedule.Schedule
}

func newScheduledCommand(hdl Handler, cmd Command, sch *schedule.Schedule) *scheduledCommand {
	return &scheduledCommand{
		hdl: hdl,
		cmd: cmd,
		sch: sch,
	}
}
