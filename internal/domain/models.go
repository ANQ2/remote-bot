package domain

import "time"

type RequestType string

const (
	RequestRemote RequestType = "remote"
	RequestSick   RequestType = "sick"
)

type DailyMode string

const (
	DailyOnline  DailyMode = "online"
	DailyOffline DailyMode = "offline"
)

type Employee struct {
	ID         int64
	TelegramID int64
	Username   string
	FullName   string
	IsPM       bool
	CreatedAt  time.Time
}

type Request struct {
	ID         int64
	EmployeeID int64
	Type       RequestType
	Date       time.Time
	DateFrom   *time.Time
	DateTo     *time.Time
	Notified   bool
	CreatedAt  time.Time
}

type Daily struct {
	ID        int64
	Date      time.Time
	Time      string
	Mode      DailyMode
	Location  string
	CreatedBy int64
	Notified  bool
	CreatedAt time.Time
}

type RequestWithEmployee struct {
	Request
	EmployeeFullName   string
	EmployeeTelegramID int64
	EmployeeUsername   string
}

type DialogStep string

const (
	StepNone            DialogStep = ""
	StepAwaitType       DialogStep = "await_type"
	StepAwaitDate       DialogStep = "await_date"
	StepAwaitSickFrom   DialogStep = "await_sick_from"
	StepAwaitSickTo     DialogStep = "await_sick_to"
	StepPMAwaitDate     DialogStep = "pm_await_date"
	StepPMAwaitTime     DialogStep = "pm_await_time"
	StepPMAwaitMode     DialogStep = "pm_await_mode"
	StepPMAwaitLocation DialogStep = "pm_await_location"
	StepPMConfirm       DialogStep = "pm_confirm"
)

type DialogState struct {
	TelegramID int64
	Step       DialogStep
	Payload    map[string]string
	UpdatedAt  time.Time
}
