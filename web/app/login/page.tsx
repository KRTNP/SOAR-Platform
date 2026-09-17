import { redirect } from "next/navigation";
import { auth, signIn } from "@/lib/auth";

// callbackUrl (set by proxy.ts's redirect to /login, see auth.ts's
// authorized callback) is normally an absolute same-origin URL like
// "http://localhost:3000/audit" — extract just the path so redirect()
// never sends the browser to a different origin.
function safeRedirectPath(callbackUrl: string | undefined): string {
  if (!callbackUrl) return "/";
  if (callbackUrl.startsWith("/")) return callbackUrl;
  try {
    return new URL(callbackUrl).pathname || "/";
  } catch {
    return "/";
  }
}

// Landing here with a session already set (a stale bookmark, the back
// button, or the one-hop bounce through /login that can follow the OIDC
// callback — see adr/0024's Consequences) used to render the sign-in
// button regardless, producing a confusing mix: Sidebar's footer already
// shows the signed-in user while the main content still offers to sign in
// again. Checking the session here and bouncing onward fixes that.
export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ callbackUrl?: string }>;
}) {
  const session = await auth();
  if (session) {
    const { callbackUrl } = await searchParams;
    redirect(safeRedirectPath(callbackUrl));
  }

  return (
    <div className="flex flex-1 flex-col items-center justify-center px-4 py-24">
      <div className="flex w-full max-w-[380px] flex-col gap-6 rounded-lg border border-border bg-surface p-8 shadow-xl">
        <div className="flex flex-col items-center gap-3 text-center">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth={1.5}
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-9 w-9 text-accent"
          >
            <path d="M12 3l7 3v5c0 4.5-3 8.5-7 10-4-1.5-7-5.5-7-10V6l7-3z" />
          </svg>
          <div className="flex flex-col gap-1">
            <h1 className="text-base font-semibold tracking-tight">
              SOAR <span className="font-normal text-text-tertiary">/ lab</span>
            </h1>
            <p className="text-sm text-text-tertiary">Sign in with your lab Keycloak account to continue.</p>
          </div>
        </div>

        <form
          action={async () => {
            "use server";
            await signIn("keycloak");
          }}
        >
          <button
            type="submit"
            className="w-full rounded-sm bg-accent px-4 py-2.5 text-sm font-medium text-bg transition-opacity hover:opacity-90"
          >
            Sign in with Keycloak
          </button>
        </form>

        <p className="text-center font-mono text-2xs tracking-wide text-text-tertiary uppercase">
          Lab environment &middot; identity provided by Keycloak
        </p>
      </div>
    </div>
  );
}
