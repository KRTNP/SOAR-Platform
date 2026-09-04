import NextAuth from "next-auth";
import Keycloak from "next-auth/providers/keycloak";

declare module "next-auth" {
  interface Session {
    accessToken?: string;
    roles?: string[];
  }
}

declare module "@auth/core/jwt" {
  interface JWT {
    accessToken?: string;
    roles?: string[];
  }
}

// adr/0024 — Auth.js owns the authorization-code+PKCE exchange and the
// session cookie so this app never hand-rolls OIDC. soar-web is a public
// client (no client secret) in deploy/compose/keycloak/realm-soar.json,
// hence token_endpoint_auth_method: "none" and explicit PKCE.
export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [
    Keycloak({
      clientId: "soar-web",
      issuer: process.env.AUTH_KEYCLOAK_ISSUER_URL,
      client: { token_endpoint_auth_method: "none" },
      checks: ["pkce"],
    }),
  ],
  session: { strategy: "jwt" },
  pages: { signIn: "/login" },
  callbacks: {
    // Copies the access token and realm roles from the Keycloak profile
    // onto our own session token once, at sign-in (account/profile are
    // only present on the request that just completed the OIDC exchange).
    async jwt({ token, account, profile }) {
      if (account) {
        token.accessToken = account.access_token;
        token.roles = (profile as { realm_access?: { roles?: string[] } } | undefined)?.realm_access?.roles ?? [];
      }
      return token;
    },
    async session({ session, token }) {
      session.accessToken = token.accessToken;
      session.roles = token.roles ?? [];
      return session;
    },
    // Gates every page proxy.ts runs on (see its matcher) — logged-in users
    // are authorized, everyone else is redirected to pages.signIn above.
    authorized({ auth }) {
      return !!auth?.user;
    },
  },
});
