package state

import (
	"fmt"

	"github.com/shikarii/spanpay/server/internal/ledger"
)

// PostingPlan describes the ledger entries required for a state transition.
type PostingPlan struct {
	Reference string
	Entries   []ledger.Entry
}

// PostingsForTransition returns the ledger entries needed when a payment
// transitions between states. Returns nil for transitions with no postings
// (e.g. PAYMENT_CREATE, AUTH_FAILURE) or idempotent replays.
func PostingsForTransition(t *Transition, amount int64, feeAmount int64) *PostingPlan {
	if t.Idempotent {
		return nil
	}

	switch t.Event {
	case EventAuthSuccess:
		return &PostingPlan{
			Reference: fmt.Sprintf("authorize:%s->%s", t.From, t.To),
			Entries: []ledger.Entry{
				{Account: ledger.AcctSettlementPending, Debit: amount, Credit: 0},
				{Account: ledger.AcctMerchantWalletPending, Debit: 0, Credit: amount},
			},
		}

	case EventCaptureSuccess:
		return &PostingPlan{
			Reference: fmt.Sprintf("capture:%s->%s", t.From, t.To),
			Entries: []ledger.Entry{
				{Account: ledger.AcctMerchantWalletPending, Debit: amount, Credit: 0},
				{Account: ledger.AcctMerchantWalletAvail, Debit: 0, Credit: amount},
				{Account: ledger.AcctSettlementAssets, Debit: amount, Credit: 0},
				{Account: ledger.AcctSettlementPending, Debit: 0, Credit: amount},
			},
		}

	case EventRefundSuccess:
		return &PostingPlan{
			Reference: fmt.Sprintf("refund:%s->%s", t.From, t.To),
			Entries: []ledger.Entry{
				{Account: ledger.AcctMerchantWalletAvail, Debit: amount, Credit: 0},
				{Account: ledger.AcctSettlementAssets, Debit: 0, Credit: amount},
			},
		}

	case EventDisputeCreated:
		entries := []ledger.Entry{
			{Account: ledger.AcctMerchantWalletAvail, Debit: amount, Credit: 0},
			{Account: ledger.AcctSettlementAssets, Debit: 0, Credit: amount},
		}
		if feeAmount > 0 {
			entries = append(entries,
				ledger.Entry{Account: ledger.AcctFeeExpense, Debit: feeAmount, Credit: 0},
				ledger.Entry{Account: ledger.AcctSettlementAssets, Debit: 0, Credit: feeAmount},
			)
		}
		return &PostingPlan{
			Reference: fmt.Sprintf("dispute:%s->%s", t.From, t.To),
			Entries:   entries,
		}

	case EventDisputeWon:
		return &PostingPlan{
			Reference: fmt.Sprintf("dispute-won:%s->%s", t.From, t.To),
			Entries: []ledger.Entry{
				{Account: ledger.AcctSettlementAssets, Debit: amount, Credit: 0},
				{Account: ledger.AcctMerchantWalletAvail, Debit: 0, Credit: amount},
			},
		}

	default:
		return nil
	}
}

// AllPostingsForTransition returns posting plans for the transition and any
// skipped intermediate transitions (from High-Water Mark forward-skip).
// Plans are returned in chronological order: skipped first, then the final event.
func AllPostingsForTransition(t *Transition, amount int64, feeAmount int64) []*PostingPlan {
	var plans []*PostingPlan
	for i := range t.Skipped {
		if p := PostingsForTransition(&t.Skipped[i], amount, 0); p != nil {
			plans = append(plans, p)
		}
	}
	if p := PostingsForTransition(t, amount, feeAmount); p != nil {
		plans = append(plans, p)
	}
	return plans
}
