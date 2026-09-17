// One module so the console speaks about time in one voice.
//
// Before this there were three coexisting formats across eight files: a bare
// `toLocaleString()` (metrics, case header, audit tooltip), a hand-built
// "M/D, h:mm" assembled from two separate calls (approvals, audit rows,
// RelativeTime), and a date-only "Sep 17, 2026" (settings, timeline
// separators) — plus two clock formats that disagreed on whether the hour was
// `numeric` or `2-digit`. An operator cross-checking an approval against the
// audit log was reading the same instant written two different ways.
//
// Four display formats, each with one job, and one machine format. Nothing
// else should call `toLocale*` directly; reach for one of these, and add a
// fifth here if none of them fits.

type When = string | Date;

const at = (w: When): Date => (typeof w === "string" ? new Date(w) : w);

// In a row, beside other content: short enough not to widen a column.
// "9/17, 2:05 PM"
export function formatTimestamp(when: When): string {
  const d = at(when);
  return `${d.toLocaleDateString([], { month: "numeric", day: "numeric" })}, ${d.toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
  })}`;
}

// A record's own timestamp, where the reader is establishing when something
// happened rather than scanning a column. "Sep 17, 2026, 2:05 PM"
export function formatPrecise(when: When): string {
  return at(when).toLocaleString([], {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

// Time of day to the second, for ordering events within a day whose date is
// already established by a separator or heading. "2:05:33 PM"
export function formatClock(when: When): string {
  return at(when).toLocaleTimeString([], { hour: "numeric", minute: "2-digit", second: "2-digit" });
}

// Date with no time. "Sep 17, 2026"
export function formatDay(when: When): string {
  return at(when).toLocaleDateString([], { year: "numeric", month: "short", day: "numeric" });
}

// The unambiguous machine form, for `title` and `dateTime` attributes. An
// operator comparing the console against a server log needs an instant with a
// zone, which none of the localised formats above carries.
export function formatIso(when: When): string {
  return at(when).toISOString();
}
