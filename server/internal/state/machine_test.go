package state

import (
	"errors"
	"testing"

	"github.com/shikarii/spanpay/server/internal/ledger"
)

func TestValidTransitions(t *testing.T) {
	cases := []struct {
		from  PaymentState
		event Event
		want  PaymentState
	}{
		{StateNone, EventPaymentCreate, StateCreated},
		{StateCreated, EventAuthSuccess, StateAuthorized},
		{StateCreated, EventAuthFailure, StateFailed},
		{StateAuthorized, EventCaptureRequest, StateProcessing},
		{StateProcessing, EventCaptureSuccess, StateCaptured},
		{StateAuthorized, EventVoidRequest, StateVoided},
		{StateCaptured, EventRefundRequest, StateRefundedPending},
		{StateRefundedPending, EventRefundSuccess, StateRefunded},
		{StateCaptured, EventDisputeCreated, StateDisputed},
		{StateDisputed, EventDisputeWon, StateCaptured},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.event), func(t *testing.T) {
			tr, err := ApplyEvent(tc.from, tc.event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tr.To != tc.want {
				t.Fatalf("got %s, want %s", tr.To, tc.want)
			}
			if tr.Idempotent {
				t.Fatal("expected non-idempotent transition")
			}
		})
	}
}

func TestInvalidTransitions(t *testing.T) {
	cases := []struct {
		from  PaymentState
		event Event
	}{
		{StateNone, EventAuthSuccess},
		{StateCreated, EventCaptureSuccess},
		{StateAuthorized, EventRefundRequest},
		{StateProcessing, EventAuthSuccess},
		{StateCaptured, EventAuthSuccess},
		{StateDisputed, EventRefundRequest},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.event), func(t *testing.T) {
			_, err := ApplyEvent(tc.from, tc.event)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("expected ErrInvalidTransition, got: %v", err)
			}
		})
	}
}

func TestTerminalStatesRejectAllEvents(t *testing.T) {
	terminals := []PaymentState{StateVoided, StateFailed, StateRefunded}
	events := []Event{
		EventPaymentCreate, EventAuthSuccess, EventAuthFailure,
		EventCaptureRequest, EventCaptureSuccess, EventVoidRequest,
		EventRefundRequest, EventRefundSuccess, EventDisputeCreated, EventDisputeWon,
	}
	for _, s := range terminals {
		for _, e := range events {
			t.Run(string(s)+"/"+string(e), func(t *testing.T) {
				_, err := ApplyEvent(s, e)
				if err == nil {
					t.Fatal("expected error from terminal state")
				}
				if !errors.Is(err, ErrTerminalState) {
					t.Fatalf("expected ErrTerminalState, got: %v", err)
				}
			})
		}
	}
}

func TestIdempotentTransitions(t *testing.T) {
	cases := []struct {
		name    string
		current PaymentState
		event   Event
	}{
		{"duplicate AUTH_SUCCESS", StateAuthorized, EventAuthSuccess},
		{"duplicate CAPTURE_SUCCESS", StateCaptured, EventCaptureSuccess},
		{"duplicate REFUND_SUCCESS", StateRefunded, EventRefundSuccess},
		{"duplicate PAYMENT_CREATE", StateCreated, EventPaymentCreate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// For REFUNDED: it's terminal, so it should return ErrTerminalState, not idempotent.
			// Let's skip that case; the terminal test covers it.
			if tc.current == StateRefunded {
				_, err := ApplyEvent(tc.current, tc.event)
				if !errors.Is(err, ErrTerminalState) {
					t.Fatalf("expected ErrTerminalState for terminal %s, got: %v", tc.current, err)
				}
				return
			}
			tr, err := ApplyEvent(tc.current, tc.event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tr.Idempotent {
				t.Fatal("expected idempotent transition")
			}
			if tr.From != tr.To {
				t.Fatalf("idempotent should not change state: from=%s to=%s", tr.From, tr.To)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	if !IsTerminal(StateVoided) {
		t.Fatal("VOIDED should be terminal")
	}
	if !IsTerminal(StateFailed) {
		t.Fatal("FAILED should be terminal")
	}
	if !IsTerminal(StateRefunded) {
		t.Fatal("REFUNDED should be terminal")
	}
	if IsTerminal(StateCaptured) {
		t.Fatal("CAPTURED should not be terminal")
	}
}

// --- Posting plan tests ---

func TestPostingsForAuthorize(t *testing.T) {
	tr := &Transition{From: StateCreated, To: StateAuthorized, Event: EventAuthSuccess}
	plan := PostingsForTransition(tr, 10000, 0)
	if plan == nil {
		t.Fatal("expected posting plan")
	}
	if len(plan.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(plan.Entries))
	}
	assertEntry(t, plan.Entries[0], ledger.AcctSettlementPending, 10000, 0)
	assertEntry(t, plan.Entries[1], ledger.AcctMerchantWalletPending, 0, 10000)
	assertBalanced(t, plan.Entries)
}

func TestPostingsForCapture(t *testing.T) {
	tr := &Transition{From: StateProcessing, To: StateCaptured, Event: EventCaptureSuccess}
	plan := PostingsForTransition(tr, 10000, 0)
	if plan == nil {
		t.Fatal("expected posting plan")
	}
	if len(plan.Entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(plan.Entries))
	}
	assertBalanced(t, plan.Entries)
}

func TestPostingsForRefund(t *testing.T) {
	tr := &Transition{From: StateRefundedPending, To: StateRefunded, Event: EventRefundSuccess}
	plan := PostingsForTransition(tr, 5000, 0)
	if plan == nil {
		t.Fatal("expected posting plan")
	}
	assertEntry(t, plan.Entries[0], ledger.AcctMerchantWalletAvail, 5000, 0)
	assertEntry(t, plan.Entries[1], ledger.AcctSettlementAssets, 0, 5000)
	assertBalanced(t, plan.Entries)
}

func TestPostingsForDisputeWithFee(t *testing.T) {
	tr := &Transition{From: StateCaptured, To: StateDisputed, Event: EventDisputeCreated}
	plan := PostingsForTransition(tr, 10000, 1500)
	if plan == nil {
		t.Fatal("expected posting plan")
	}
	if len(plan.Entries) != 4 {
		t.Fatalf("expected 4 entries (reversal + fee), got %d", len(plan.Entries))
	}
	assertBalanced(t, plan.Entries)
}

func TestPostingsForDisputeWon(t *testing.T) {
	tr := &Transition{From: StateDisputed, To: StateCaptured, Event: EventDisputeWon}
	plan := PostingsForTransition(tr, 10000, 0)
	if plan == nil {
		t.Fatal("expected posting plan")
	}
	assertEntry(t, plan.Entries[0], ledger.AcctSettlementAssets, 10000, 0)
	assertEntry(t, plan.Entries[1], ledger.AcctMerchantWalletAvail, 0, 10000)
	assertBalanced(t, plan.Entries)
}

func TestPostingsNilForIdempotent(t *testing.T) {
	tr := &Transition{From: StateCaptured, To: StateCaptured, Event: EventCaptureSuccess, Idempotent: true}
	if PostingsForTransition(tr, 10000, 0) != nil {
		t.Fatal("idempotent transitions should produce no postings")
	}
}

func TestPostingsNilForNoLedgerEvents(t *testing.T) {
	tr := &Transition{From: StateNone, To: StateCreated, Event: EventPaymentCreate}
	if PostingsForTransition(tr, 0, 0) != nil {
		t.Fatal("PAYMENT_CREATE should produce no postings")
	}

	tr2 := &Transition{From: StateCreated, To: StateFailed, Event: EventAuthFailure}
	if PostingsForTransition(tr2, 0, 0) != nil {
		t.Fatal("AUTH_FAILURE should produce no postings")
	}
}

func assertEntry(t *testing.T, e ledger.Entry, account string, debit, credit int64) {
	t.Helper()
	if e.Account != account || e.Debit != debit || e.Credit != credit {
		t.Fatalf("entry mismatch: got {%s %d %d}, want {%s %d %d}",
			e.Account, e.Debit, e.Credit, account, debit, credit)
	}
}

func assertBalanced(t *testing.T, entries []ledger.Entry) {
	t.Helper()
	if err := ledger.ValidateBalanced(entries); err != nil {
		t.Fatalf("entries not balanced: %v", err)
	}
}
