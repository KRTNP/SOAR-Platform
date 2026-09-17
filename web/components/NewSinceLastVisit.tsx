"use client";

import { useEffect, useSyncExternalStore } from "react";

const KEY = "soar:last-seen";

// Every SOC tool surveyed answers "what should I look at" with a global
// lifecycle status (New / In progress / Closed) that any user's action
// mutates. That is correct for a shift-based team with handover, and it
// cannot answer *this* operator's actual question, because they are the only
// user and their sessions are episodic: they want to know what changed while
// they were away.
//
// With exactly one operator a per-viewer watermark is trivially correct, so
// this keeps one in localStorage. It is a per-viewer convenience, never
// shared state and never read back by the server: if storage is unavailable
// the component renders nothing and the queue is unaffected.
function safeGet(): string | null {
  try {
    return localStorage.getItem(KEY);
  } catch {
    return null;
  }
}

// Frozen for the lifetime of the page load. The effect below writes a fresh
// watermark immediately, and without freezing, the banner would recompute
// against the value it just wrote and always say "nothing new".
let frozen: string | null | undefined;
function getFrozenSnapshot(): string | null {
  if (frozen === undefined) frozen = safeGet();
  return frozen;
}

const subscribe = () => () => {};
const getServerSnapshot = (): string | null => null;

// Floor, not round: rounding made a 35-minute-old visit report "1h ago" and
// skip the sub-hour branch entirely.
function sinceLabel(iso: string): string {
  const hours = Math.floor((Date.now() - new Date(iso).getTime()) / 3_600_000);
  if (hours < 1) return "less than an hour ago";
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return days === 1 ? "yesterday" : `${days} days ago`;
}

// Lets a row say "this one" instead of making the operator carry a count
// down the list. Same frozen watermark the banner reads, so the two can never
// disagree.
export function useIsNewSinceLastVisit(iso: string): boolean {
  const lastSeen = useSyncExternalStore(subscribe, getFrozenSnapshot, getServerSnapshot);
  if (!lastSeen) return false;
  const cutoff = new Date(lastSeen).getTime();
  return !Number.isNaN(cutoff) && new Date(iso).getTime() > cutoff;
}

export function NewMarker({ iso }: { iso: string }) {
  const isNew = useIsNewSinceLastVisit(iso);
  if (!isNew) return null;
  return (
    <span
      className="inline-flex shrink-0 items-center gap-1 rounded-sm border border-accent/50 bg-accent/10 px-1.5 py-px text-2xs font-medium text-accent"
      title="Changed since your last visit"
    >
      new
    </span>
  );
}

export function NewSinceLastVisit({ timestamps }: { timestamps: string[] }) {
  const lastSeen = useSyncExternalStore(subscribe, getFrozenSnapshot, getServerSnapshot);

  useEffect(() => {
    try {
      localStorage.setItem(KEY, new Date().toISOString());
    } catch {
      // Private mode or blocked storage — the watermark simply doesn't persist.
    }
  }, []);

  // No watermark yet (first ever visit, or storage unavailable): say nothing
  // rather than claiming everything is new.
  if (!lastSeen) return null;

  const cutoff = new Date(lastSeen).getTime();
  if (Number.isNaN(cutoff)) return null;

  const count = timestamps.filter((t) => new Date(t).getTime() > cutoff).length;
  if (count === 0) {
    return (
      <p className="text-xs text-text-tertiary">
        Nothing new since your last visit, {sinceLabel(lastSeen)}.
      </p>
    );
  }

  return (
    <p className="text-xs text-text-secondary">
      <span className="font-medium text-accent">
        {count} case{count === 1 ? "" : "s"} new or updated
      </span>{" "}
      since your last visit, {sinceLabel(lastSeen)}.
    </p>
  );
}
