package helper

import "regexp"

// RouteEventReason
const (
	FailedCreateRoute  = "CreateRouteFailed"
	FailedSyncRoute    = "SyncRouteFailed"
	SucceedCreateRoute = "CreatedRoute"
)

var re = regexp.MustCompile(".*(Message:.*)")

func GetLogMessage(err error) string {
	if err == nil {
		return ""
	}
	var message string
	sub := re.FindStringSubmatch(err.Error())
	if len(sub) > 1 {
		message = sub[1]
	} else {
		message = err.Error()
	}
	return message
}

// EventType type of event associated with an informer
type EventType string

const (
	// CreateEvent event associated with new objects in an informer
	CreateEvent EventType = "CREATE"
	// UpdateEvent event associated with an object update in an informer
	UpdateEvent EventType = "UPDATE"
	// ConfigurationEvent event associated when a controller configuration object is created or updated
	ConfigurationEvent EventType = "CONFIGURATION"
)

// Event holds the context of an event.
type Event struct {
	Type EventType
	Obj  interface{}
}
