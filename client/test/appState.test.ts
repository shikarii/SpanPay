import { describe, expect, it } from "vitest";

import { buildCapabilityCards } from "../src/appState";

describe("appState", () => {
  it("exposes the core orchestration capabilities in a stable order", () => {
    expect(buildCapabilityCards().map((card) => card.title)).toEqual([
      "Immutable ledger",
      "Provider adapters",
      "Inbox and outbox",
      "Reconciliation",
    ]);
  });
});
