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
];

// Below md this recomposes into a compact top bar (logo + nav only — the
// tenant/connector/dry-run status block is secondary information and is
// dropped rather than shrunk, per gridgeist's "recompose, don't shrink"
// rule) so the app is usable on a phone instead of rendering a squeezed
// 236px column beside the content. md and up keeps the original side rail.
export function Sidebar({ connectorTarget, connectorVersion }: { connectorTarget?: string; connectorVersion?: string }) {
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

      <div className="hidden flex-col gap-2 border-t border-border pt-3.5 md:flex">
        <span className="px-2.5 font-mono text-[10.5px] tracking-wider text-text-tertiary">
          TENANT &middot; LAB
        </span>
        {connectorTarget && (
          <span
            title={connectorVersion}
            className={`flex items-center gap-1.5 rounded-[4px] border px-2.5 py-1.5 text-[11.5px] ${
              isMockConnector ? "border-border-strong text-text-secondary" : "border-danger/30 bg-danger/[0.06] text-danger"
            }`}
          >
            <span className={`h-[7px] w-[7px] shrink-0 rounded-full ${isMockConnector ? "bg-text-tertiary" : "bg-danger"}`} />
            <span className="truncate">{isMockConnector ? "mock connector" : `live: ${connectorTarget}`}</span>
          </span>
        )}
        {/* This used to be a hardcoded "dry-run mode" label — accurate back
            when no real connector existed at all, but actively misleading
            once a real one (isMockConnector false) can execute for real:
            it sat right next to the danger-styled "live: X" badge above,
            each contradicting the other. An approval is still required
            either way (adr/0011); only the honesty of what happens *after*
            approval changes here. */}
        <span
          className={`flex items-center gap-1.5 rounded-[4px] border px-2.5 py-1.5 text-[11.5px] ${
            isMockConnector ? "border-warn/25 bg-warn/[0.08] text-warn" : "border-danger/30 bg-danger/[0.08] text-danger"
          }`}
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth={1.7}
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-3 w-3 shrink-0"
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
          {isMockConnector ? "dry-run mode" : "live actions — real effects"}
        </span>
      </div>
    </aside>
  );
}
