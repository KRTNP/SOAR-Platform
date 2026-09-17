"use client";

import { useEffect } from "react";

// An operator's whole session here is "decide 1-8 things and leave"
// (PRODUCT.md). Before this the only way through a queue was the mouse, or Tab
// through every link and button inside every row — the usability review scored
// keyboard efficiency 1 out of 40.
//
// The rules are deliberately global and attribute-driven rather than lifted
// into each page's state: a row opts in with `data-kbd-row`, a decision button
// with `data-kbd-approve` / `data-kbd-reject`, and a filter field with
// `data-kbd-filter`. A new list gets the shortcuts by adding one attribute,
// which is the only version of this that survives the next page being added.
const ROW = "[data-kbd-row]";

function isTypingTarget(el: EventTarget | null): boolean {
  if (!(el instanceof HTMLElement)) return false;
  if (el.isContentEditable) return true;
  return el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.tagName === "SELECT";
}

export function KeyboardShortcuts() {
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      // Never shadow a browser or OS chord, and never eat a keystroke meant
      // for a text field — including the rejection-reason input, which is the
      // one place `r` is a letter someone is trying to type.
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      if (isTypingTarget(event.target)) return;
      // A modal owns the keyboard while it is open; it has its own Escape and
      // Enter handling and its own focus trap.
      if (document.querySelector('[role="dialog"]')) return;

      // Every list here renders twice — a desktop table and a stack of mobile
      // cards — and CSS hides one of them. Both are in the DOM, so without
      // this filter `j` would walk through rows nobody can see.
      const rows = Array.from(document.querySelectorAll<HTMLElement>(ROW)).filter(
        (row) => row.getClientRects().length > 0,
      );
      const active = document.activeElement as HTMLElement | null;
      const current = rows.findIndex((row) => row === active || row.contains(active));

      if (event.key === "j" || event.key === "k") {
        if (rows.length === 0) return;
        event.preventDefault();
        // From nowhere, either key enters the list at the top rather than
        // doing nothing — the first press should always have an effect.
        const next =
          current === -1
            ? 0
            : event.key === "j"
              ? Math.min(current + 1, rows.length - 1)
              : Math.max(current - 1, 0);
        rows[next].focus();
        rows[next].scrollIntoView({ block: "nearest" });
        return;
      }

      if (event.key === "a" || event.key === "r") {
        if (current === -1) return;
        const button = rows[current].querySelector<HTMLButtonElement>(
          event.key === "a" ? "[data-kbd-approve]" : "[data-kbd-reject]",
        );
        // A decided row has no buttons and a busy one has disabled buttons.
        // Both mean the key does nothing, rather than throwing focus around.
        if (!button || button.disabled) return;
        event.preventDefault();
        button.click();
        return;
      }

      if (event.key === "/") {
        const filter = document.querySelector<HTMLElement>("[data-kbd-filter]");
        if (!filter) return;
        event.preventDefault();
        filter.focus();
      }
    }

    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  return null;
}

// Shortcuts nobody is told about score the same as shortcuts that do not
// exist, so every surface that opts in also says so. `kbd` rather than plain
// text because these are keys, and the distinction survives being read aloud.
export function ShortcutHint({ keys }: { keys: [string, string][] }) {
  return (
    <p className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-text-tertiary">
      {keys.map(([key, meaning]) => (
        <span key={key} className="inline-flex items-center gap-1.5">
          <kbd className="rounded-sm border border-border-strong bg-surface-raised px-1.5 py-0.5 font-mono text-2xs text-text-secondary">
            {key}
          </kbd>
          {meaning}
        </span>
      ))}
    </p>
  );
}
