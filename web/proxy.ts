export { auth as proxy } from "@/lib/auth";

// Next.js 16 renamed the middleware/proxy file convention (this file must
// be named proxy.ts, not middleware.ts, for Next to run it at all — see
// node_modules/next/dist/docs/.../file-conventions/proxy.md's "Migration
// to Proxy" section). Everything except the OIDC callback route and static
// assets is gated behind auth.ts's callbacks.authorized.
export const config = {
  matcher: ["/((?!api/auth|_next/static|_next/image|favicon.ico|login).*)"],
};
