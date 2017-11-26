package rules

type EventType int8

const (
	EventTypeError EventType = iota
	EventTypeTrigger
)

type RulesEvent struct {
	EventType EventType
	Msg interface{}
}