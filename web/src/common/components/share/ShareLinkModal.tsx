import {
  ActionIcon,
  Center,
  CopyButton,
  Modal,
  Stack,
  Text,
  TextInput,
  Tooltip,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconCheck, IconCopy } from "@tabler/icons-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { ROUTES } from "@/common/routes/routes";
import { buildShareUrl } from "@/common/lib/url";
import { FullscreenQrOverlay } from "./FullscreenQrOverlay";
import { QrCode } from "./QrCode";

interface ShareLinkModalProps {
  opened: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  url: string;
  /** Words people can type at /code instead of the URL. */
  code?: string | null;
  /** Extra content below the QR code, e.g. expiry or revoke actions. */
  children?: ReactNode;
}

export function ShareLinkModal({
  opened,
  onClose,
  title,
  description,
  url,
  code,
  children,
}: ShareLinkModalProps) {
  const { t } = useTranslation("common");
  const [fullscreen, { open: openFullscreen, close: closeFullscreen }] =
    useDisclosure(false);

  return (
    <>
      <Modal opened={opened} onClose={onClose} title={title} size="md">
        <Stack gap="md">
          {description && <Text size="sm">{description}</Text>}
          <TextInput
            readOnly
            value={url}
            onFocus={(e) => e.currentTarget.select()}
            styles={{ input: { fontFamily: "monospace" } }}
            rightSection={
              <CopyButton value={url}>
                {({ copied, copy }) => (
                  <Tooltip
                    label={copied ? t("share.copied") : t("share.copyLink")}
                  >
                    <ActionIcon
                      variant="subtle"
                      color={copied ? "green" : "gray"}
                      onClick={copy}
                      aria-label={t("share.copyLink")}
                    >
                      {copied ? (
                        <IconCheck size={16} />
                      ) : (
                        <IconCopy size={16} />
                      )}
                    </ActionIcon>
                  </Tooltip>
                )}
              </CopyButton>
            }
          />
          {code && (
            <Stack gap={2} align="center">
              <Text size="xs" c="dimmed">
                {t("share.codeHint", { url: buildShareUrl(ROUTES.CODE) })}
              </Text>
              <Text ff="monospace" fw={700} size="xl" ta="center">
                {code}
              </Text>
            </Stack>
          )}
          <Center>
            <UnstyledButton
              onClick={openFullscreen}
              aria-label={t("share.qrHint")}
            >
              <Stack gap={4} align="center">
                <QrCode value={url} size={260} />
                <Text size="xs" c="dimmed">
                  {t("share.qrHint")}
                </Text>
              </Stack>
            </UnstyledButton>
          </Center>
          {children}
        </Stack>
      </Modal>
      <FullscreenQrOverlay
        opened={fullscreen}
        onClose={closeFullscreen}
        url={url}
        code={code}
      />
    </>
  );
}
