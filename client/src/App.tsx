import { buildCapabilityCards } from "./appState";

export function App() {
  const cards = buildCapabilityCards();

  return (
    <main style={{ minHeight: "100vh", padding: "40px", background: "#f3f7fb", color: "#132238", fontFamily: "system-ui, sans-serif" }}>
      <section style={{ maxWidth: "960px", margin: "0 auto" }}>
        <p style={{ textTransform: "uppercase", letterSpacing: "0.12em", fontSize: "12px", color: "#4c6b8a" }}>SpanPay</p>
        <h1 style={{ margin: "12px 0 8px", fontSize: "42px" }}>Provider-agnostic payment orchestration</h1>
        <p style={{ maxWidth: "720px", lineHeight: 1.6, color: "#36526f" }}>
          SpanPay is being scaffolded as a sovereign orchestration service that owns payment state, immutable ledger postings,
          idempotency, webhook durability, and reconciliation instead of outsourcing truth to providers.
        </p>

        <div style={{ display: "grid", gap: "16px", gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", marginTop: "28px" }}>
          {cards.map((card) => (
            <article key={card.title} style={{ padding: "18px", borderRadius: "16px", background: "#ffffff", boxShadow: "0 12px 24px rgba(19, 34, 56, 0.08)" }}>
              <h2 style={{ margin: "0 0 8px", fontSize: "18px" }}>{card.title}</h2>
              <p style={{ margin: 0, lineHeight: 1.5, color: "#4c6b8a" }}>{card.detail}</p>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
