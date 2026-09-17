"use client";

import { ProblemBanner } from "@/components/ProblemBanner";

// Settings makes three independent server calls (tenants, allowlist,
// critical targets); any one of them throwing used to take out the whole
// app shell via the root boundary instead of failing inside this route.
export default function SettingsError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex flex-col gap-3">
      <ProblemBanner fallbackTitle="Something went wrong loading settings" />
      <button
        onClick={reset}
        className="w-fit rounded-sm border border-border-strong px-3 py-1.5 text-sm text-text-secondary hover:bg-surface-hover"
      >
        Try again
      </button>
    </div>
  );
}
