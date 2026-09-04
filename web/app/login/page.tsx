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
    <div className="flex flex-1 flex-col items-center justify-center gap-4 py-24 text-center">
      <div className="flex flex-col gap-1.5">
        <h1 className="text-lg font-semibold tracking-tight">SOAR Console</h1>
        <p className="text-[13px] text-text-tertiary">Sign in with your lab Keycloak account to continue.</p>
      </div>
      <form
        action={async () => {
          "use server";
          await signIn("keycloak");
        }}
      >
        <button
          type="submit"
          className="rounded-[4px] bg-accent px-4 py-2 text-[13px] font-medium text-bg transition-opacity hover:opacity-90"
        >
          Sign in with Keycloak
        </button>
      </form>
    </div>
  );
}
