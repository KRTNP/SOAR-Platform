import { signIn } from "@/lib/auth";

export default function LoginPage() {
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
