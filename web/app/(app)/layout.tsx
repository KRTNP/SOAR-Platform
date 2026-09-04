import { redirect } from "next/navigation";
import { Sidebar } from "@/components/Sidebar";
import { getMetrics } from "@/lib/api-client/client";
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

  // Best-effort: the sidebar's connector badge is a convenience, not load-
  // bearing, so a metrics fetch failure (backend not running, tenant with no
  // data yet) just omits the badge rather than breaking every page's layout.
  let connectorTarget: string | undefined;
  let connectorVersion: string | undefined;
  try {
    const snap = await getMetrics();
    connectorTarget = snap.connector_target;
    connectorVersion = snap.connector_version;
  } catch {
    // omit the badge
  }

  return (
    <div className="flex min-h-screen flex-1 flex-col md:flex-row">
      <Sidebar
        connectorTarget={connectorTarget}
        connectorVersion={connectorVersion}
        userEmail={session.user?.email ?? undefined}
        onSignOut={signOutAction}
      />
      <div className="flex min-w-0 flex-1 flex-col overflow-y-auto px-4 pt-5 pb-8 md:px-[34px] md:pt-[26px]">{children}</div>
    </div>
  );
}
