import { useState } from "react";
import type { MailSettings } from "@/api/types";
import type { ServerSettingsState } from "./useServerSettings";

/** The mail server as the admin types it; the password only when typed. */
export interface MailForm {
  host: string;
  port: number;
  user: string;
  from: string;
  consentCopyEmail: string;
  password: string;
}

const formOf = (m?: MailSettings): MailForm => ({
  host: m?.host ?? "",
  port: m?.port || 587,
  user: m?.user ?? "",
  from: m?.from ?? "",
  consentCopyEmail: m?.consentCopyEmail ?? "",
  password: "",
});

/**
 * The site's mail server, edited as a whole and saved at once. The password
 * is never shown: an empty field keeps it, and clearing it is its own step.
 */
export function useMailSettingsForm({ settings, update }: ServerSettingsState) {
  const stored = settings?.mail;
  // What the admin typed; the rest is what is stored.
  const [edits, setEdits] = useState<Partial<MailForm>>({});
  const form: MailForm = { ...formOf(stored), ...edits };

  const set = <K extends keyof MailForm>(key: K, value: MailForm[K]) => setEdits((e) => ({ ...e, [key]: value }));
  const dirty = JSON.stringify(form) !== JSON.stringify(formOf(stored));
  const saved = { onSuccess: () => setEdits({}) };

  const fields = () => ({
    host: form.host.trim(),
    port: form.port,
    user: form.user.trim(),
    from: form.from.trim(),
    consentCopyEmail: form.consentCopyEmail.trim(),
  });
  const save = () =>
    update.mutate({ mail: { ...fields(), ...(form.password ? { password: form.password } : {}) } }, saved);
  const clearPassword = () => update.mutate({ mail: { ...fields(), password: "" } }, saved);

  return {
    form,
    set,
    dirty,
    save,
    clearPassword,
    passwordSet: !!stored?.passwordSet,
    saving: update.isPending,
    failed: update.isError,
  };
}
