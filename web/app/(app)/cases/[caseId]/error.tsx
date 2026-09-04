"use client";

import Link from "next/link";
import { ProblemBanner } from "@/components/ProblemBanner";

export default function CaseDetailError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex flex-col gap-3">
      <ProblemBanner fallbackTitle="Something went wrong loading this case" />
      <div className="flex gap-2">
        <button
          onClick={reset}
          className="w-fit rounded-[4px] border border-border-strong px-3 py-1.5 text-[13px] text-text-secondary hover:bg-surface-hover"
        >
          Try again
        </button>
        <Link
          href="/"
          className="w-fit rounded-[4px] border border-border-strong px-3 py-1.5 text-[13px] text-text-secondary hover:bg-surface-hover"
        >
          Back to queue
        </Link>
      </div>
    </div>
  );
}
