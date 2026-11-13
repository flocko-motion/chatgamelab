import {
  Alert,
  Button,
  CopyButton,
  Group,
  Modal,
  Stack,
  Text,
  Textarea,
  TextInput,
} from "@mantine/core";
import {
  IconCheck,
  IconCopy,
  IconInfoCircle,
  IconMail,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { RoutesInstitutionApplicationLinkResponse } from "@/api/generated";

interface ApplicationLinkModalProps {
  result: RoutesInstitutionApplicationLinkResponse | null;
  onClose: () => void;
}

function CopyAction({ value }: { value: string }) {
  const { t } = useTranslation("common");
  return (
    <CopyButton value={value} timeout={2000}>
      {({ copied, copy }) => (
        <Button
          size="xs"
          variant="light"
          color={copied ? "green" : undefined}
          leftSection={
            copied ? <IconCheck size={14} /> : <IconCopy size={14} />
          }
          onClick={copy}
        >
          {copied
            ? t("admin.applications.link.copied")
            : t("admin.applications.link.copy")}
        </Button>
      )}
    </CopyButton>
  );
}

/**
 * Shows a freshly created confirmation link as a ready-to-send mail (recipient, subject, text),
 * so an admin can send it by hand from the ChatGameLab mailbox while the server sends no mails.
 */
export function ApplicationLinkModal({
  result,
  onClose,
}: ApplicationLinkModalProps) {
  const { t, i18n } = useTranslation("common");
  const to = result?.to ?? "";
  const subject = result?.subject ?? "";
  const body = result?.body ?? "";
  const expires = result?.application?.expiresAt
    ? new Date(result.application.expiresAt).toLocaleString(i18n.language, {
        dateStyle: "short",
        timeStyle: "short",
      })
    : "";
  const mailto = `mailto:${encodeURIComponent(to)}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`;

  return (
    <Modal
      opened={!!result}
      onClose={onClose}
      title={t("admin.applications.link.title")}
      size="lg"
    >
      <Stack gap="md">
        <Alert
          color={result?.mailConfigured ? "gray" : "blue"}
          icon={<IconInfoCircle size={16} />}
        >
          {result?.mailConfigured
            ? t("admin.applications.link.configured")
            : t("admin.applications.link.manual")}
        </Alert>
        <Group align="flex-end" gap="xs" wrap="nowrap">
          <TextInput
            label={t("admin.applications.link.to")}
            value={to}
            readOnly
            style={{ flex: 1 }}
          />
          <CopyAction value={to} />
        </Group>
        <Group align="flex-end" gap="xs" wrap="nowrap">
          <TextInput
            label={t("admin.applications.link.subject")}
            value={subject}
            readOnly
            style={{ flex: 1 }}
          />
          <CopyAction value={subject} />
        </Group>
        <Stack gap={4}>
          <Group justify="space-between" align="flex-end">
            <Text size="sm" fw={500}>
              {t("admin.applications.link.body")}
            </Text>
            <CopyAction value={body} />
          </Group>
          <Textarea value={body} readOnly autosize minRows={8} maxRows={16} />
        </Stack>
        {expires && (
          <Text size="xs" c="dimmed">
            {t("admin.applications.link.validity", { date: expires })}
          </Text>
        )}
        <Group justify="flex-end" gap="xs">
          <Button variant="subtle" onClick={onClose}>
            {t("admin.applications.link.close")}
          </Button>
          <Button
            component="a"
            href={mailto}
            leftSection={<IconMail size={16} />}
          >
            {t("admin.applications.link.openMail")}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
