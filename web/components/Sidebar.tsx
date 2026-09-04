"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

function NavIcon({ children }: { children: ReactNode }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-[17px] w-[17px] shrink-0 text-text-tertiary group-aria-[current=page]:text-accent"
    >
      {children}
    </svg>
  );
}

const NAV_ITEMS = [
  {
    href: "/",
    label: "Queue",
    icon: (
      <>
        <line x1="4" y1="7" x2="20" y2="7" />
        <line x1="4" y1="12" x2="20" y2="12" />
        <line x1="4" y1="17" x2="14" y2="17" />
      </>
    ),
  },
  {
    href: "/approvals",
    label: "Approvals",
    icon: (
      <>
        <polyline points="4 12.5 9 17.5 20 6" />
      </>
    ),
  },
  {
    href: "/metrics",
    label: "Metrics",
    icon: (
      <>
        <line x1="5" y1="19" x2="5" y2="11" />
        <line x1="12" y1="19" x2="12" y2="5" />
        <line x1="19" y1="19" x2="19" y2="14" />
      </>
    ),
  },
  {
    href: "/audit",
    label: "Audit",
    icon: (
      <>
        <rect x="5" y="4" width="14" height="16" rx="1.5" />
        <polyline points="8.5 11.5 11 14 15.5 8.5" />
      </>
    ),
  },
  {
    href: "/search",
    label: "Search",
    icon: (
      <>
        <circle cx="10.5" cy="10.5" r="6" />
        <line x1="15" y1="15" x2="19.5" y2="19.5" />
      </>
    ),
  },
  {
    href: "/settings",
    label: "Settings",
    icon: (
      <>
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1.08-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1.08 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
      </>
    ),
  },
];

// Below md this recomposes into a compact top bar (logo + nav only — the
// tenant/connector/dry-run status block is secondary information and is
// dropped rather than shrunk, per gridgeist's "recompose, don't shrink"
// rule) so the app is usable on a phone instead of rendering a squeezed
// 236px column beside the content. md and up keeps the original side rail.
export function Sidebar({
  connectorTarget,
  connectorVersion,
  userEmail,
  onSignOut,
}: {
  connectorTarget?: string;
  connectorVersion?: string;
  userEmail?: string;
  onSignOut?: () => Promise<void>;
}) {
  const pathname = usePathname();
  const isMockConnector = connectorVersion?.includes("mock") ?? true;

  return (
    <aside className="flex w-full shrink-0 flex-row items-center gap-3 border-b border-border bg-surface p-[10px_16px] md:w-[236px] md:flex-col md:items-stretch md:gap-0 md:border-r md:border-b-0 md:p-[22px_16px]">
      <div className="flex items-center gap-2.5 px-0 md:mb-[22px] md:px-2 md:pt-1">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.6}
          strokeLinecap="round"
          strokeLinejoin="round"
          className="h-5 w-5 shrink-0 text-accent"
        >
          <path d="M12 3l7 3v5c0 4.5-3 8.5-7 10-4-1.5-7-5.5-7-10V6l7-3z" />
        </svg>
        <span className="hidden text-sm font-semibold tracking-tight sm:inline">
          SOAR <span className="font-normal text-text-tertiary">/ lab</span>
        </span>
      </div>

      <nav className="flex flex-1 flex-row gap-0.5 md:flex-col">
        {NAV_ITEMS.map((item) => {
          const current = pathname === item.href || (item.href !== "/" && pathname.startsWith(item.href));
          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={current ? "page" : undefined}
              className={`group flex items-center gap-3 rounded-[4px] px-2.5 py-2.5 text-[13px] font-medium ${
                current ? "bg-surface-raised text-text-primary" : "text-text-secondary"
              }`}
            >
              <NavIcon>{item.icon}</NavIcon>
              <span className="hidden md:inline">{item.label}</span>
            </Link>
          );
        })}
      </nav>

      <div className="hidden flex-col gap-3 border-t border-border pt-3.5 md:flex">
        {/* One status line, not two near-duplicate badges: mock vs. live is
            a single fact (dry-run vs. real effects on a named target), so
            it gets one visual treatment instead of two stacked pills that
            used to repeat the same "is this real" signal in slightly
            different words. */}
        <div
          className={`flex items-start gap-2 rounded-[4px] border px-2.5 py-2 text-[11.5px] ${
            isMockConnector ? "border-border-strong text-text-secondary" : "border-danger/30 bg-danger/[0.07] text-danger"
          }`}
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth={1.7}
            strokeLinecap="round"
            strokeLinejoin="round"
            className="mt-0.5 h-3 w-3 shrink-0"
          >
            {isMockConnector ? (
              <>
                <rect x="5" y="10" width="14" height="9" rx="1.5" />
                <path d="M8 10V7a4 4 0 0 1 8 0v3" />
              </>
            ) : (
              <>
                <path d="M12 3l7 3v5c0 4.5-3 8.5-7 10-4-1.5-7-5.5-7-10V6l7-3z" />
                <path d="M12 8v5" />
                <path d="M12 16.5v.01" />
              </>
            )}
          </svg>
          <div className="flex min-w-0 flex-col">
            <span className="font-medium">{isMockConnector ? "Dry-run mode" : "Live — real effects"}</span>
            {connectorTarget && <span className="truncate text-text-tertiary">{connectorTarget}</span>}
          </div>
        </div>

        {onSignOut ? (
          <form action={onSignOut} className="flex items-center justify-between gap-2 border-t border-border pt-3">
            <div className="flex min-w-0 items-center gap-2">
              <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-raised font-mono text-[10px] font-medium text-text-secondary">
                {(userEmail ?? "L")[0].toUpperCase()}
              </span>
              <div className="flex min-w-0 flex-col leading-tight">
                <span className="truncate text-[11.5px] text-text-secondary">{userEmail ?? "lab"}</span>
                <span className="font-mono text-[10px] tracking-wider text-text-tertiary uppercase">tenant &middot; lab</span>
              </div>
            </div>
            <button
              type="submit"
              className="shrink-0 rounded-[4px] px-1.5 py-1 text-[11px] font-medium text-text-tertiary hover:bg-surface-raised hover:text-text-primary"
            >
              Sign out
            </button>
          </form>
        ) : (
          <span className="font-mono text-[10.5px] tracking-wider text-text-tertiary uppercase">tenant &middot; lab</span>
        )}
      </div>
    </aside>
  );
}
