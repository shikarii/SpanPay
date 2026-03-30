package state

import (
	"errors"
	"fmt"
)

// PaymentState represents the current status of a payment.
type PaymentState string

const (
	StateNone            PaymentState = ""
	StateCreated         PaymentState = "CREATED"
	StateAuthorized      PaymentState = "AUTHORIZED"
	StateProcessing      PaymentState = "PROCESSING"
	StateCaptured        PaymentState = "CAPTURED"
	StateVoided          PaymentState = "VOIDED"
	StateFailed          PaymentState = "FAILED"
	StateRefundedPending PaymentState = "REFUNDED_PENDING"
	StateRefunded        PaymentState = "REFUNDED"
	StateDisputed        PaymentState = "DISPUTED"
)

// Event represents a trigger that may cause a state transition.
type Event string

const (
	EventPaymentCreate  Event = "PAYMENT_CREATE"
	EventAuthSuccess    Event = "AUTH_SUCCESS"
	EventAuthFailure    Event = "AUTH_FAILURE"
	EventCaptureRequest Event = "CAPTURE_REQUEST"
	EventCaptureSuccess Event = "CAPTURE_SUCCESS"
	EventVoidRequest    Event = "VOID_REQUEST"
	EventRefundRequest  Event = "REFUND_REQUEST"
	EventRefundSuccess  Event = "REFUND_SUCCESS"
	EventDisputeCreated Event = "DISPUTE_CREATED"
	EventDisputeWon     Event = "DISPUTE_WON"
)

var (
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrTerminalState     = errors.New("payment is in a terminal state")
)

// Transition describes the outcome of applying an event.
type Transition struct {
	From       PaymentState
	To         PaymentState
	Event      Event
	Idempotent bool
	Skipped    []Transition // intermediate transitions auto-applied via High-Water Mark
}

type key struct {
	from  PaymentState
	event Event
}

var terminalStates = map[PaymentState]bool{
	StateVoided:   true,
	StateFailed:   true,
	StateRefunded: true,
}

var table = map[key]PaymentState{
	{StateNone, EventPaymentCreate}:            StateCreated,
	{StateCreated, EventAuthSuccess}:           StateAuthorized,
	{StateCreated, EventAuthFailure}:           StateFailed,
	{StateAuthorized, EventCaptureRequest}:     StateProcessing,
	{StateProcessing, EventCaptureSuccess}:     StateCaptured,
	{StateAuthorized, EventVoidRequest}:        StateVoided,
	{StateCaptured, EventRefundRequest}:        StateRefundedPending,
	{StateRefundedPending, EventRefundSuccess}: StateRefunded,
	{StateCaptured, EventDisputeCreated}:       StateDisputed,
	{StateDisputed, EventDisputeWon}:           StateCaptured,
}

// happyPath defines the canonical event chain on the main payment lifecycle.
// Used by the High-Water Mark logic to compute intermediate transitions
// when webhook events arrive before earlier synchronous responses commit.
var happyPath = []struct {
	event Event
	from  PaymentState
	to    PaymentState
}{
	{EventPaymentCreate, StateNone, StateCreated},
	{EventAuthSuccess, StateCreated, StateAuthorized},
	{EventCaptureRequest, StateAuthorized, StateProcessing},
	{EventCaptureSuccess, StateProcessing, StateCaptured},
}

// stateIndex maps happy-path states to their ordinal position.
var stateIndex = map[PaymentState]int{
	StateNone:       0,
	StateCreated:    1,
	StateAuthorized: 2,
	StateProcessing: 3,
	StateCaptured:   4,
}

// ApplyEvent attempts to transition from current state via event.
// Returns the transition result, or an error if the transition is invalid.
//
// High-Water Mark: if the event implies a state ahead of current on the happy
// path, intermediate transitions are auto-applied and returned in Skipped.
//
// Idempotent: if the payment is already in the target state for this event,
// the transition is accepted with Idempotent=true and no side effects.
func ApplyEvent(current PaymentState, event Event) (*Transition, error) {
	if terminalStates[current] {
		return nil, fmt.Errorf("%w: state %s rejects all events", ErrTerminalState, current)
	}

	// Direct transition.
	target, ok := table[key{current, event}]
	if ok {
		return &Transition{From: current, To: target, Event: event}, nil
	}

	// Idempotent check: if any transition with this event targets the current state,
	// the payment already completed this step.
	for k, t := range table {
		if k.event == event && t == current {
			return &Transition{From: current, To: current, Event: event, Idempotent: true}, nil
		}
	}

	// High-Water Mark: try forward-skip on the happy path.
	if tr, err := tryForwardSkip(current, event); err == nil {
		return tr, nil
	}

	return nil, fmt.Errorf("%w: no transition from %s on %s", ErrInvalidTransition, current, event)
}

// tryForwardSkip checks if the event's normal source state is ahead of current
// on the happy path. If so, it builds the intermediate transitions needed to
// reach that source, then applies the event.
func tryForwardSkip(current PaymentState, event Event) (*Transition, error) {
	currentIdx, onPath := stateIndex[current]
	if !onPath {
		return nil, ErrInvalidTransition
	}

	// Find the event's normal source state from the transition table.
	var normalSource PaymentState
	var found bool
	for k := range table {
		if k.event == event {
			normalSource = k.from
			found = true
			break
		}
	}
	if !found {
		return nil, ErrInvalidTransition
	}

	sourceIdx, sourceOnPath := stateIndex[normalSource]
	if !sourceOnPath || sourceIdx <= currentIdx {
		return nil, ErrInvalidTransition
	}

	// Build intermediate transitions from current to normalSource.
	var skipped []Transition
	for i := currentIdx; i < sourceIdx; i++ {
		step := happyPath[i]
		skipped = append(skipped, Transition{
			From:  step.from,
			To:    step.to,
			Event: step.event,
		})
	}

	finalTarget := table[key{normalSource, event}]
	return &Transition{
		From:    current,
		To:      finalTarget,
		Event:   event,
		Skipped: skipped,
	}, nil
}

// IsTerminal returns true if the state rejects all further events.
func IsTerminal(s PaymentState) bool {
	return terminalStates[s]
}
