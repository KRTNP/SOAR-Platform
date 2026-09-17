"use client";

import { useState, useTransition } from "react";
import { createTenantAction, addAllowlistEntryAction, addCriticalTargetAction } from "./actions";

const inputClass =
  "rounded-sm border border-border-strong bg-surface-raised px-2.5 py-1.5 text-sm text-text-primary placeholder:text-text-tertiary";
const buttonClass =
  "rounded-sm bg-accent px-3 py-1.5 text-sm font-medium text-bg whitespace-nowrap disabled:cursor-not-allowed disabled:opacity-40";

// The only one of the three forms with special behavior: the create
// response's api_key is never shown again after this — the list below only
// ever shows id/name/created_at. Kept in this component's own local state,
// never sent anywhere else, so a page refresh loses it exactly as
// permanently as the backend does.
export function TenantForm() {
  const [name, setName] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);
  const [createdKey, setCreatedKey] = useState<{ name: string; apiKey: string } | null>(null);

  function submit() {
    if (!name.trim()) return;
    setError(null);
    startTransition(async () => {
      try {
        const { tenant, error: problem } = await createTenantAction(name.trim());
        if (problem || !tenant) {
          setError(problem ?? "Could not create tenant. Try again.");
          return;
        }
        setCreatedKey({ name: tenant.name, apiKey: tenant.api_key ?? "" });
        setName("");
      } catch {
        setError("Could not create tenant. Try again.");
      }
    });
  }

  return (
    <div className="flex flex-col gap-2">
      {createdKey && (
        <div className="rounded-md border border-warn/30 bg-warn/[0.08] p-3 text-xs">
          <p className="font-medium text-text-primary">
            API key for &ldquo;{createdKey.name}&rdquo; &mdash; copy it now, it will not be shown again:
          </p>
          <p className="mt-1 break-all rounded-sm bg-surface-raised px-2 py-1 font-mono text-xs text-accent">
            {createdKey.apiKey}
          </p>
        </div>
      )}
      <div className="flex gap-2">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Tenant name"
          aria-label="Tenant name"
          className={`flex-1 ${inputClass}`}
        />
        <button onClick={submit} disabled={!name.trim() || isPending} className={buttonClass}>
          {isPending ? "Creating…" : "Create tenant"}
        </button>
      </div>
      {error && <p className="text-xs text-danger">{error}</p>}
    </div>
  );
}

export function AllowlistForm() {
  const [value, setValue] = useState("");
  const [reason, setReason] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  function submit() {
    if (!value.trim() || !reason.trim()) return;
    setError(null);
    startTransition(async () => {
      try {
        const { error: problem } = await addAllowlistEntryAction(value.trim(), reason.trim(), expiresAt);
        if (problem) {
          setError(problem);
          return;
        }
        setValue("");
        setReason("");
        setExpiresAt("");
      } catch {
        setError("Could not add trusted source. Try again.");
      }
    });
  }

  const dirty = value.trim() !== "" || reason.trim() !== "";

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-2">
        <input
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="Value (e.g. IP address)"
          aria-label="Trusted source value"
          className={`min-w-[180px] flex-1 ${inputClass}`}
        />
        <input
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Reason"
          aria-label="Reason"
          className={`min-w-[180px] flex-1 ${inputClass}`}
        />
        <input
          type="datetime-local"
          value={expiresAt}
          onChange={(e) => setExpiresAt(e.target.value)}
          aria-label="Expires at (optional)"
          className={inputClass}
        />
        <button onClick={submit} disabled={!value.trim() || !reason.trim() || isPending} className={buttonClass}>
          {isPending ? "Adding…" : "Add trusted source"}
        </button>
      </div>
      {dirty && !expiresAt && <p className="text-2xs text-text-tertiary">No expiry set — this entry never expires.</p>}
      {error && <p className="text-xs text-danger">{error}</p>}
    </div>
  );
}

export function CriticalTargetForm() {
  const [value, setValue] = useState("");
  const [reason, setReason] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  function submit() {
    if (!value.trim() || !reason.trim()) return;
    setError(null);
    startTransition(async () => {
      try {
        const { error: problem } = await addCriticalTargetAction(value.trim(), reason.trim());
        if (problem) {
          setError(problem);
          return;
        }
        setValue("");
        setReason("");
      } catch {
        setError("Could not add critical target. Try again.");
      }
    });
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-2">
        <input
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="Value (user or host)"
          aria-label="Critical target value"
          className={`min-w-[180px] flex-1 ${inputClass}`}
        />
        <input
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Reason"
          aria-label="Reason"
          className={`min-w-[180px] flex-1 ${inputClass}`}
        />
        <button onClick={submit} disabled={!value.trim() || !reason.trim() || isPending} className={buttonClass}>
          {isPending ? "Adding…" : "Add critical target"}
        </button>
      </div>
      {error && <p className="text-xs text-danger">{error}</p>}
    </div>
  );
}
