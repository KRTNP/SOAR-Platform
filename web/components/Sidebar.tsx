"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  ChartColumn,
  Check,
  CircleAlert,
  ClipboardCheck,
  Inbox,
  Lock,
  Search,
  Settings,
  Shield,
  ShieldAlert,
  type LucideIcon,
} from "lucide-react";

const NAV_ITEMS: { href: string; label: string; Icon: LucideIcon }[] = [
  { href: "/", label: "Queue", Icon: Inbox },
  { href: "/approvals", label: "Approvals", Icon: Check },
  { href: "/metrics", label: "Metrics", Icon: ChartColumn },
  { href: "/audit", label: "Audit", Icon: ClipboardCheck },
  { href: "/search", label: "Search", Icon: Search },
  { href: "/settings", label: "Settings", Icon: Settings },
];

// Below md this recomposes into a compact top bar (logo + nav only — the
// tenant/connector/dry-run status block is secondary information and is
// dropped rather than shrunk, per gridgeist's "recompose, don't shrink"
// rule) so the app is usable on a phone instead of rendering a squeezed
// 236px column beside the content. md and up keeps the original side rail.

// Three states, not two. "Dry-run" and "unknown" used to collapse into the
// same reassuring badge via `?? true`, so an unreachable backend rendered a
// confident claim that responses were simulated — the single fact this
// console may never guess, because it is what an operator checks before
// authorizing a write to a real firewall.
type ConnectorState = "live" | "dry_run" | "unknown";

function connectorState(version: string | undefined, reachable: boolean): ConnectorState {
  if (!reachable || version === undefined) return "unknown";
  return version.includes("mock") ? "dry_run" : "live";
}

export function Sidebar({
  connectorTarget,
  connectorVersion,
  connectorReachable = true,
  pendingApprovals,
  userEmail,
  onSignOut,
}: {
  connectorTarget?: string;
  connectorVersion?: string;
  connectorReachable?: boolean;
  pendingApprovals?: number;
  userEmail?: string;
  onSignOut?: () => Promise<void>;
}) {
  const pathname = usePathname();
  const state = connectorState(connectorVersion, connectorReachable);
  // Locked = dry-run (nothing can escape), shield-with-alert = live effects,
  // circle-with-alert = we do not know. Three icons for the three states, so
  // the shape carries the same information as the colour.
  const StatusIcon = state === "dry_run" ? Lock : state === "live" ? ShieldAlert : CircleAlert;

  return (
    <aside className="flex w-full shrink-0 flex-row items-center gap-3 border-b border-border bg-surface px-4 py-2.5 md:w-[236px] md:flex-col md:items-stretch md:gap-0 md:border-r md:border-b-0 md:px-4 md:py-5">
      <div className="flex items-center gap-2.5 px-0 md:mb-5 md:px-2 md:pt-1">
        <Shield className="h-5 w-5 shrink-0 text-accent" aria-hidden="true" />
        <span className="hidden text-sm font-semibold tracking-tight sm:inline">
          SOAR <span className="font-normal text-text-tertiary">/ lab</span>
        </span>
      </div>

      <nav className="flex flex-1 flex-row gap-0.5 md:flex-col">
        {NAV_ITEMS.map(({ href, label, Icon }) => {
          const current = pathname === href || (href !== "/" && pathname.startsWith(href));
          const waiting = href === "/approvals" && pendingApprovals !== undefined && pendingApprovals > 0;
          return (
            <Link
              key={href}
              href={href}
              aria-current={current ? "page" : undefined}
              className={`group relative flex items-center gap-3 rounded-sm px-2.5 py-2.5 text-sm font-medium ${
                current ? "bg-surface-raised text-text-primary" : "text-text-secondary"
              }`}
            >
              <Icon
                className="h-4 w-4 shrink-0 text-text-tertiary group-aria-[current=page]:text-accent"
                aria-hidden="true"
              />
              {/* The label is the accessible name on every breakpoint; below
                  md it is visually hidden rather than removed, so the compact
                  top bar is still six named links instead of six mystery
                  icons to a screen reader. */}
              <span className="sr-only md:not-sr-only md:inline">{label}</span>
              {waiting && (
                <span
                  className="ml-auto inline-flex h-4.5 min-w-4.5 items-center justify-center rounded-full bg-warn px-1.5 font-mono text-2xs font-semibold text-bg max-md:absolute max-md:top-0.5 max-md:right-0 max-md:ml-0"
                  aria-label={`${pendingApprovals} awaiting your decision`}
                >
                  {pendingApprovals}
                </span>
              )}
            </Link>
          );
        })}
      </nav>

      {/* Below md the full status block is dropped for space — but "are my
          actions real?" is not secondary information when the approve path is
          still reachable on this breakpoint. The two states that change what a
          tap means keep a compact marker; dry-run, the safe default, does not
          need one. */}
      {state !== "dry_run" && (
        <span
          className={`ml-auto inline-flex shrink-0 items-center gap-1.5 rounded-sm border px-2 py-1 text-2xs font-medium md:hidden ${
            state === "live" ? "border-danger/40 bg-danger/[0.07] text-danger" : "border-warn/40 bg-warn/[0.07] text-warn"
          }`}
        >
          <span
            aria-hidden="true"
            className={`h-1.5 w-1.5 rounded-full ${state === "live" ? "bg-danger" : "bg-warn"}`}
          />
          {state === "live" ? "Live" : "Unknown"}
        </span>
      )}

      <div className="hidden flex-col gap-3 border-t border-border pt-3.5 md:flex">
        {/* One status line, not two near-duplicate badges: mock vs. live is
            a single fact (dry-run vs. real effects on a named target), so
            it gets one visual treatment instead of two stacked pills. */}
        <div
          className={`flex items-start gap-2 rounded-sm border px-2.5 py-2 text-2xs ${
            state === "live"
              ? "border-danger/30 bg-danger/[0.07] text-danger"
              : state === "dry_run"
                ? "border-border-strong text-text-secondary"
                : "border-warn/40 bg-warn/[0.07] text-warn"
          }`}
        >
          <StatusIcon className="mt-0.5 h-3 w-3 shrink-0" aria-hidden="true" />
          <div className="flex min-w-0 flex-col">
            <span className="font-medium">
              {state === "live" ? "Live — real effects" : state === "dry_run" ? "Dry-run mode" : "Connector state unknown"}
            </span>
            {state === "unknown" ? (
              <span className="text-warn/80">Metrics unreachable — verify before approving</span>
            ) : (
              connectorTarget && <span className="truncate text-text-tertiary">{connectorTarget}</span>
            )}
          </div>
        </div>

        {onSignOut ? (
          <form action={onSignOut} className="flex items-center justify-between gap-2 border-t border-border pt-3">
            <div className="flex min-w-0 items-center gap-2">
              <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-raised font-mono text-2xs font-medium text-text-secondary">
                {(userEmail ?? "L")[0].toUpperCase()}
              </span>
              <div className="flex min-w-0 flex-col leading-tight">
                <span className="truncate text-2xs text-text-secondary">{userEmail ?? "lab"}</span>
                <span className="font-mono text-2xs tracking-wider text-text-tertiary uppercase">tenant &middot; lab</span>
              </div>
            </div>
            <button
              type="submit"
              className="shrink-0 rounded-sm px-1.5 py-1 text-2xs font-medium text-text-tertiary hover:bg-surface-raised hover:text-text-primary"
            >
              Sign out
            </button>
          </form>
        ) : (
          <span className="font-mono text-2xs tracking-wider text-text-tertiary uppercase">tenant &middot; lab</span>
        )}
      </div>
    </aside>
  );
}
