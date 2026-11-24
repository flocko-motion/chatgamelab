import { useState } from "react";
import { useCreateApiKey, usePlatforms } from "@/api/hooks";
import type { KeyShareView } from "@/api/types";
import { useAuth } from "@/providers/AuthProvider";

interface NewKeyErrors {
  name?: string;
  platform?: string;
  key?: string;
}

/** A key name that says whose key it is, for which platform, and since when. */
function defaultKeyName(platformName: string, userName: string | undefined): string {
  const now = new Date();
  const date = [now.getDate(), now.getMonth() + 1]
    .map((n) => String(n).padStart(2, "0"))
    .concat(String(now.getFullYear()).slice(-2))
    .join(".");
  return [userName, platformName, date].filter(Boolean).join(" ");
}

/**
 * The form that adds a key of the caller's: the platforms that take keys, the
 * platform, name and secret, their errors, and the submit that stores the key.
 * The key always becomes the caller's own; what happens next (a check, a share)
 * is up to the page that holds the form.
 */
export function useNewKeyForm(t: (key: string) => string) {
  const { backendUser } = useAuth();
  const { data: platforms } = usePlatforms();
  const keyPlatforms = (platforms ?? [])
    .filter((p) => p.userKeys)
    .map((p) => ({ id: p.id, name: p.name || p.id, info: p }));

  const createApiKey = useCreateApiKey();
  const [name, setName] = useState("");
  const [platform, setPlatform] = useState("");
  const [key, setKey] = useState("");
  const [errors, setErrors] = useState<NewKeyErrors>({});

  const choosePlatform = (id: string) => {
    setPlatform(id);
    const platformName = keyPlatforms.find((p) => p.id === id)?.name ?? id;
    setName(defaultKeyName(platformName, backendUser?.name));
  };

  /** Empties the form, with the first platform chosen. */
  const reset = () => {
    setKey("");
    setErrors({});
    if (keyPlatforms[0]) choosePlatform(keyPlatforms[0].id);
  };

  /** Stores the key; the caller's own-use share of it, or null when the form or the server refused. */
  const submit = async (): Promise<KeyShareView | null> => {
    const found: NewKeyErrors = {};
    if (!name.trim()) found.name = t("apiKeys.errors.nameRequired");
    if (!platform) found.platform = t("apiKeys.errors.platformRequired");
    if (!key.trim()) found.key = t("apiKeys.errors.keyRequired");
    setErrors(found);
    if (Object.keys(found).length > 0) return null;
    try {
      return await createApiKey.mutateAsync({ name, platform, key });
    } catch {
      // shown by the mutation's error handler
      return null;
    }
  };

  return {
    /** Whether any platform takes keys, so a key can be added at all. */
    available: keyPlatforms.length > 0,
    platforms: keyPlatforms,
    tiers: keyPlatforms.find((p) => p.id === platform)?.info.tiers ?? [],
    name,
    setName,
    platform,
    setPlatform: choosePlatform,
    key,
    setKey,
    errors,
    reset,
    submit,
    loading: createApiKey.isPending,
  };
}

export type NewKeyFormState = ReturnType<typeof useNewKeyForm>;
