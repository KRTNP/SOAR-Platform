// Shared by the table row and the mobile card so both truncate identically —
// full case-id + full-precision timestamps were what pushed the desktop
// table wide enough to clip the Decide column's Reject button past a
// standard 1440px viewport (see approval-actions.tsx's usage).

// Keeps the tail, not the head. These ids are ULIDs: the leading characters
// are a timestamp prefix shared by everything created in the same period, so
// truncating from the right threw away the only part that tells two rows
// apart and left a column of near-identical strings.
export function truncateCaseId(id: string): string {
  return id.length > 22 ? `…${id.slice(-19)}` : id;
}

export function formatExpiry(iso: string): string {
  const d = new Date(iso);
  return `${d.toLocaleDateString([], { month: "numeric", day: "numeric" })}, ${d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })}`;
}
