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
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex w-[236px] shrink-0 flex-col border-r border-border bg-surface p-[22px_16px]">
      <div className="mb-[22px] flex items-center gap-2.5 px-2 pt-1">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.6}
          strokeLinecap="round"
          strokeLinejoin="round"
          className="h-5 w-5 text-accent"
        >
          <path d="M12 3l7 3v5c0 4.5-3 8.5-7 10-4-1.5-7-5.5-7-10V6l7-3z" />
        </svg>
        <span className="text-sm font-semibold tracking-tight">
          SOAR <span className="font-normal text-text-tertiary">/ lab</span>
        </span>
      </div>

      <nav className="flex flex-1 flex-col gap-0.5">
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
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="flex flex-col gap-2 border-t border-border pt-3.5">
        <span className="px-2.5 font-mono text-[10.5px] tracking-wider text-text-tertiary">
          TENANT &middot; LAB
        </span>
        <span className="flex items-center gap-1.5 rounded-[4px] border border-warn/25 bg-warn/[0.08] px-2.5 py-1.5 text-[11.5px] text-warn">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth={1.7}
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-3 w-3 shrink-0"
          >
            <rect x="5" y="10" width="14" height="9" rx="1.5" />
            <path d="M8 10V7a4 4 0 0 1 8 0v3" />
          </svg>
          dry-run mode
        </span>
      </div>
    </aside>
  );
}
