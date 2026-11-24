import { ActionIcon, Badge, Divider, Loader, SimpleGrid, Stack, TextInput, Tooltip } from "@mantine/core";
import { IconArrowBackUp, IconBrush } from "@tabler/icons-react";
import { Fragment } from "react";
import { useTranslation } from "react-i18next";
import { Label } from "@components/typography";
import { BRANDING_GROUPS, useSiteBrandingForm, type BrandingField } from "../hooks/useSiteBrandingForm";
import { SettingsCard } from "./SettingsCard";

/** How the site presents itself (name, operator, links), for the admin. */
export function SiteBranding() {
  const { t } = useTranslation("common");
  const form = useSiteBrandingForm();

  return (
    <SettingsCard
      icon={<IconBrush size={20} />}
      title={t("serverSettings.branding.title")}
      description={t("serverSettings.branding.description")}
      busy={form.saving}
    >
      {form.loading ? (
        <Loader size="sm" />
      ) : (
        BRANDING_GROUPS.map((group, i) => (
          <Fragment key={group.id}>
            {i > 0 && <Divider />}
            <Stack gap="sm">
              <Label>{t(`serverSettings.branding.groups.${group.id}`)}</Label>
              <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md" verticalSpacing="sm">
                {group.fields.map((field) => (
                  <BrandingInput key={field} field={field} form={form} />
                ))}
              </SimpleGrid>
            </Stack>
          </Fragment>
        ))
      )}
    </SettingsCard>
  );
}

/** One branding field: saved when it loses focus; a changed one can go back to its default. */
function BrandingInput({
  field,
  form,
}: {
  field: BrandingField;
  form: ReturnType<typeof useSiteBrandingForm>;
}) {
  const { t } = useTranslation("common");
  const value = form.value(field);
  const changed = !form.isDefault(field);
  const error = form.error(field);

  return (
    <TextInput
      key={`${field}-${value}`}
      label={
        <>
          {t(`serverSettings.branding.fields.${field}`)}
          {changed && (
            <Badge ml="xs" size="xs" variant="light" color="accent">
              {t("serverSettings.branding.changed")}
            </Badge>
          )}
        </>
      }
      placeholder={t("serverSettings.branding.hidden")}
      error={error && t(`serverSettings.branding.errors.${error}`)}
      defaultValue={value}
      type={field === "supportEmail" ? "email" : field.endsWith("Url") ? "url" : "text"}
      size="sm"
      disabled={form.saving}
      onBlur={(e) => form.save(field, e.currentTarget.value)}
      onKeyDown={(e) => {
        if (e.key === "Enter") e.currentTarget.blur();
      }}
      rightSection={
        changed && (
          <Tooltip label={t("serverSettings.branding.resetTo", { value: form.defaultOf(field) })} withArrow>
            <ActionIcon
              variant="subtle"
              color="gray"
              size="sm"
              aria-label={t("serverSettings.branding.reset")}
              onClick={() => form.reset(field)}
            >
              <IconArrowBackUp size={16} />
            </ActionIcon>
          </Tooltip>
        )
      }
    />
  );
}
