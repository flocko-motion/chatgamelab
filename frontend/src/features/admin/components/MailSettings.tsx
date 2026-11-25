import { Alert, Group, NumberInput, PasswordInput, SimpleGrid, TextInput } from "@mantine/core";
import { IconAlertCircle, IconMail } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import type { ServerSettingsState } from "../hooks/useServerSettings";
import { useMailSettingsForm } from "../hooks/useMailSettingsForm";
import { SettingsCard } from "./SettingsCard";

/** The mail server the site's mails go through, for the admin. */
export function MailSettings({ state }: { state: ServerSettingsState }) {
  const { t } = useTranslation("common");
  const m = useMailSettingsForm(state);
  const k = (key: string) => t(`serverSettings.mail.${key}`);

  return (
    <SettingsCard icon={<IconMail size={20} />} title={k("title")} description={k("description")} busy={m.saving}>
      <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md" verticalSpacing="sm">
        <TextInput
          label={k("host")}
          description={k("hostHint")}
          placeholder="smtp.example.org"
          value={m.form.host}
          onChange={(e) => m.set("host", e.currentTarget.value)}
          size="sm"
        />
        <NumberInput
          label={k("port")}
          description={k("portHint")}
          value={m.form.port}
          onChange={(v) => m.set("port", typeof v === "number" ? v : Number(v) || 587)}
          min={1}
          max={65535}
          allowDecimal={false}
          allowNegative={false}
          size="sm"
        />
        <TextInput
          label={k("user")}
          value={m.form.user}
          onChange={(e) => m.set("user", e.currentTarget.value)}
          autoComplete="off"
          size="sm"
        />
        <PasswordInput
          label={k("password")}
          description={m.passwordSet ? k("passwordSet") : k("passwordNone")}
          value={m.form.password}
          onChange={(e) => m.set("password", e.currentTarget.value)}
          autoComplete="new-password"
          size="sm"
        />
        <TextInput
          label={k("from")}
          description={k("fromHint")}
          placeholder={k("fromPlaceholder")}
          value={m.form.from}
          onChange={(e) => m.set("from", e.currentTarget.value)}
          size="sm"
        />
        <TextInput
          label={k("consentCopy")}
          description={k("consentCopyHint")}
          type="email"
          value={m.form.consentCopyEmail}
          onChange={(e) => m.set("consentCopyEmail", e.currentTarget.value)}
          size="sm"
        />
      </SimpleGrid>
      {m.failed && (
        <Alert color="red" icon={<IconAlertCircle size={16} />}>
          {k("saveFailed")}
        </Alert>
      )}
      <Group justify="space-between">
        {m.passwordSet ? (
          <TextButton onClick={m.clearPassword} disabled={m.saving}>
            {k("clearPassword")}
          </TextButton>
        ) : (
          <span />
        )}
        <ActionButton onClick={m.save} disabled={!m.dirty} loading={m.saving}>
          {k("save")}
        </ActionButton>
      </Group>
    </SettingsCard>
  );
}
