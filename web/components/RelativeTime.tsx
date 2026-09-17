"use client";

import { useSyncExternalStore } from "react";

function relative(iso: string): string {
  const minutes = Math.round((Date.now() - new Date(iso).getTime()) / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

function subscribe(onChange: () => void) {
  const id = setInterval(onChange, 60_000);
  return () => clearInterval(id);
}

// Bucketed to the minute so a re-render only happens when the label could
// actually change.
const getSnapshot = () => Math.floor(Date.now() / 60_000);
const getServerSnapshot = () => null;

// "3m ago" computed during a server render is wrong the moment it arrives and
// disagrees with what the client would compute. useSyncExternalStore is the
// supported way to say "the server has no value for this": it renders the
// absolute time server-side, swaps to relative on hydration, and ticks once a
// minute after that. The absolute value stays in title/dateTime either way,
// which is what an operator cross-checking against a log actually needs.
export function RelativeTime({ iso, className }: { iso: string; className?: string }) {
  const minuteBucket = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

  const label =
    minuteBucket === null
      ? new Date(iso).toLocaleString([], { month: "numeric", day: "numeric", hour: "numeric", minute: "2-digit" })
      : relative(iso);

  return (
    <time dateTime={iso} title={new Date(iso).toISOString()} className={className}>
      {label}
    </time>
  );
}
