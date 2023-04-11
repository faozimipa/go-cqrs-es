package events

import "github.com/faozimipa/go-cqrs-es/pkg/es"

const (
	EmailChangedEventType es.EventType = "EMAIL_CHANGED_V1"
)

type EmailChangedEventV1 struct {
	Email    string `json:"email"`
	Metadata []byte `json:"-"`
}
