import { redirect } from "next/navigation";
import { Sidebar } from "@/components/Sidebar";
import { getMetrics, listApprovals } from "@/lib/api-client/client";
import { auth } from "@/lib/auth";
import { signOutAction } from "../actions";

// Every route in this group requires a session — proxy.ts already redirects
// an unauthenticated request to /login before it gets here, but this is the
// same check made explicit at the layout level rather than trusted solely
// from the edge (defense in depth, and the only way this layout can safely
// assume session.user exists below).
export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const session = await auth();
  if (!session) {
    redirect("/login");
  }

  // A metrics fetch failure must not break every page's layout — but it also
  // must not be silently indistinguishable from a successful read. The
  // sidebar's connector line states whether responses run dry or for real,
  // and guessing that is the one thing it may never do, so the failure is
  // passed through as an explicit "unreachable" rather than swallowed.
  // Fetched together so the shell renders once. The pending-approval count
  // is the one number in this app worth carrying in the chrome: sessions are
  // episodic (PRODUCT.md), so "something is blocked on you" has to survive
  // whichever page you happen to land on. Undefined means "not known" and
  // renders nothing, rather than a reassuring zero.
  const [metrics, approvals] = await Promise.allSettled([getMetrics(), listApprovals()]);

  const connectorReachable = metrics.status === "fulfilled";
  const connectorTarget = metrics.status === "fulfilled" ? metrics.value.connector_target : undefined;
  const connectorVersion = metrics.status === "fulfilled" ? metrics.value.connector_version : undefined;
  const pendingApprovals =
    approvals.status === "fulfilled"
      ? approvals.value.items.filter((a) => a.status === "pending").length
      : undefined;

  return (
    <div className="flex min-h-screen flex-1 flex-col md:flex-row">
      <Sidebar
        connectorTarget={connectorTarget}
        connectorVersion={connectorVersion}
        connectorReachable={connectorReachable}
        pendingApprovals={pendingApprovals}
        userEmail={session.user?.email ?? undefined}
        onSignOut={signOutAction}
      />
      <div className="flex min-w-0 flex-1 flex-col overflow-y-auto px-4 pt-5 pb-8 md:px-8 md:pt-6">{children}</div>
    </div>
  );
}
