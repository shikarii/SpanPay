export interface CapabilityCard {
  title: string;
  detail: string;
}

export function buildCapabilityCards(): CapabilityCard[] {
  return [
    {
      title: "Immutable ledger",
      detail: "Balances derive from append-only double-entry postings instead of mutable counters.",
    },
    {
      title: "Provider adapters",
      detail: "PSP-specific APIs stay behind normalized orchestration boundaries.",
    },
    {
      title: "Inbox and outbox",
      detail: "Webhook durability and async side effects are modeled as first-class seams.",
    },
    {
      title: "Reconciliation",
      detail: "External settlement data is compared against internal truth instead of trusted blindly.",
    },
  ];
}
