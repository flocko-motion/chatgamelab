import {
  ActionIcon,
  Box,
  Button,
  CopyButton,
  Group,
  Modal,
  Stack,
  Text,
  Title,
  Tooltip,
} from "@mantine/core";
import { IconCheck, IconCopy } from "@tabler/icons-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { QrCode } from "./QrCode";

interface FullscreenQrOverlayProps {
  opened: boolean;
  onClose: () => void;
  url: string;
  title: string;
  icon: ReactNode;
  /** Mantine colour of the title bar. */
  color: "blue" | "green";
  description?: string;
  /** Shown below the link, e.g. the expiry date. */
  children?: ReactNode;
  /** Buttons placed before "Schließen". */
  actions?: ReactNode;
}

/** A link as QR code for the projector; closes only through its button or Esc. */
export function FullscreenQrOverlay({
  opened,
  onClose,
  url,
  title,
  icon,
  color,
  description,
  children,
  actions,
}: FullscreenQrOverlayProps) {
  const { t } = useTranslation("common");

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      fullScreen
      withCloseButton={false}
      closeOnClickOutside={false}
      padding={0}
      styles={{
        content: {
          background: "#ffffff",
          color: "#000000",
          display: "flex",
          flexDirection: "column",
        },
        body: { flex: 1, display: "flex", flexDirection: "column" },
      }}
    >
      <Group
        gap="sm"
        wrap="nowrap"
        px="lg"
        py="md"
        c="white"
        style={{ background: `var(--mantine-color-${color}-filled)` }}
      >
        {icon}
        <Title order={2} c="white" style={{ fontSize: "clamp(1.1rem, 2.5vw, 1.8rem)" }}>
          {title}
        </Title>
      </Group>
      <Stack align="center" justify="center" gap="lg" p="md" style={{ flex: 1 }}>
        {description && (
          <Text
            c="#333333"
            ta="center"
            style={{ fontSize: "clamp(1rem, 2.2vw, 1.5rem)" }}
          >
            {description}
          </Text>
        )}
        <QrCode value={url} size="min(55vh, 80vw)" />
        <Group gap="xs" justify="center" wrap="nowrap" maw="100%">
          <Text
            ff="monospace"
            fw={700}
            ta="center"
            c="#000000"
            style={{
              fontSize: "clamp(1rem, 3vw, 2.2rem)",
              wordBreak: "break-all",
            }}
          >
            {url}
          </Text>
          <CopyButton value={url}>
            {({ copied, copy }) => (
              <Tooltip label={copied ? t("share.copied") : t("share.copyLink")}>
                <ActionIcon
                  variant="subtle"
                  color={copied ? "green" : "gray"}
                  onClick={copy}
                  aria-label={t("share.copyLink")}
                >
                  {copied ? <IconCheck size={18} /> : <IconCopy size={18} />}
                </ActionIcon>
              </Tooltip>
            )}
          </CopyButton>
        </Group>
        {children}
      </Stack>
      <Box px="lg" pb="lg">
        <Group justify="center" gap="md">
          {actions}
          <Button onClick={onClose} variant="default" size="md" radius="md">
            {t("close")}
          </Button>
        </Group>
      </Box>
    </Modal>
  );
}
