package v2

import "fmt"

type OneApiError struct {
	ErrorMessage ErrorMessage `json:"error"`
}

func (e *OneApiError) Error() string {
	return e.String()
}

type ErrorMessage struct {
	Code        string       `json:"code"`
	Message     string       `json:"message"`
	Description string       `json:"@message"`
	Type        string       `json:"@type"`
	Context     ErrorContext `json:"@context"`
	IsPermanent bool         `json:"@permanent"`
}

type ErrorContext struct {
	Timestamp        string `json:"timestamp"`
	ServiceAlias     string `json:"serviceAlias"`
	MachineName      string `json:"machineName"`
	ProcessName      string `json:"processName"`
	ProcessId        int    `json:"processId"`
	ThreadId         int    `json:"threadId"`
	ClientRequestId  string `json:"clientRequestId"`
	ActivityId       string `json:"activityId"`
	SubActivityId    string `json:"subActivityId"`
	ActivityType     string `json:"activityType"`
	ParentActivityId string `json:"parentActivityId"`
	ActivityStack    string `json:"activityStack"`
}

func (e *OneApiError) String() string {
	return fmt.Sprintf("OneApiError(Error=%#v)", e.ErrorMessage)
}

func (e *ErrorMessage) String() string {
	return fmt.Sprintf("ErrorMessage(Code=%s, Message=%s, Type=%s, ErrorContext=%v, IsPermanent=%t)", e.Code, e.Message, e.Type, e.Context, e.IsPermanent)
}

func (e *ErrorContext) String() string {
	return fmt.Sprintf("ErrorContext(Timestamp=%s, ServiceAlias=%s, MachineName=%s, ProcessName=%s, ProcessId=%d, ThreadId=%d, ClientRequestId=%s, ActivityId=%s, SubActivityId=%s, ActivityType=%s, ParentActivityId=%s, ActivityStack=%s)", e.Timestamp, e.ServiceAlias, e.MachineName, e.ProcessName, e.ProcessId, e.ThreadId, e.ClientRequestId, e.ActivityId, e.SubActivityId, e.ActivityType, e.ParentActivityId, e.ActivityStack)
}
