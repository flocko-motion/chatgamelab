import { Select, Stack, TextInput } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { HelperText } from "@components/typography";
import { getAiQualityTierLabel } from "@/common/lib/aiQualityTier";
import type { NewKeyFormState } from "../hooks/useNewKeyForm";

/** The fields that add a key: platform (with the models its tiers use), name and secret. */
export function NewKeyFields({ form }: { form: NewKeyFormState }) {
  const { t } = useTranslation("common");
  return (
    <Stack gap="md">
      <Select
        label={t("apiKeys.createModal.platformLabel")}
        placeholder={t("apiKeys.createModal.platformPlaceholder")}
        data={form.platforms.map((p) => ({ value: p.id, label: p.name }))}
        required
        value={form.platform || null}
        onChange={(value) => value && form.setPlatform(value)}
        error={form.errors.platform}
        allowDeselect={false}
      />
      {form.tiers.length > 0 && (
        <Stack gap={0}>
          {form.tiers.map((tier) => (
            <HelperText key={tier.id}>
              {getAiQualityTierLabel(t, tier.id ?? "")}: {tier.models?.story?.name}
            </HelperText>
          ))}
        </Stack>
      )}
      <TextInput
        label={t("apiKeys.createModal.nameLabel")}
        placeholder={t("apiKeys.createModal.namePlaceholder")}
        value={form.name}
        onChange={(e) => form.setName(e.currentTarget.value)}
        error={form.errors.name}
      />
      <TextInput
        label={t("apiKeys.createModal.keyLabel")}
        placeholder={t("apiKeys.createModal.keyPlaceholder")}
        description={t("apiKeys.createModal.keyDescription")}
        required
        type="password"
        value={form.key}
        onChange={(e) => form.setKey(e.currentTarget.value)}
        error={form.errors.key}
      />
    </Stack>
  );
}
