// Shared by the table row and the mobile card so both truncate identically —
// full case-id + full-precision timestamps were what pushed the desktop
// table wide enough to clip the Decide column's Reject button past a
// standard 1440px viewport (see approval-actions.tsx's usage).

export function truncateCaseId(id: string): string {
  return id.length > 22 ? `${id.slice(0, 19)}…` : id;
}

export function formatExpiry(iso: string): string {
  const d = new Date(iso);
  return `${d.toLocaleDateString([], { month: "numeric", day: "numeric" })}, ${d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })}`;
}
