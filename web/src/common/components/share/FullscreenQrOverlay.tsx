import { Modal, Stack, Text } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { ROUTES } from "@/common/routes/routes";
import { buildShareUrl } from "@/common/lib/url";
import { QrCode } from "./QrCode";

interface FullscreenQrOverlayProps {
  opened: boolean;
  onClose: () => void;
  url: string;
  /** Words to read off the projector; falls back to the URL. */
  code?: string | null;
}

export function FullscreenQrOverlay({
  opened,
  onClose,
  url,
  code,
}: FullscreenQrOverlayProps) {
  const { t } = useTranslation("common");

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      fullScreen
      withCloseButton={false}
      padding={0}
      styles={{
        content: { background: "#ffffff", color: "#000000" },
        body: { height: "100%" },
      }}
    >
      <Stack
        align="center"
        justify="center"
        gap="lg"
        h="100%"
        p="md"
        onClick={onClose}
        style={{ cursor: "pointer" }}
      >
        <QrCode value={url} size="min(60vh, 80vw)" />
        {code && (
          <Text
            c="#333333"
            ta="center"
            style={{ fontSize: "clamp(1rem, 2.5vw, 1.8rem)" }}
          >
            {t("share.codeHint", { url: buildShareUrl(ROUTES.CODE) })}
          </Text>
        )}
        {code && (
          <Text
            ff="monospace"
            fw={700}
            ta="center"
            c="#000000"
            style={{ fontSize: "clamp(1.5rem, 6vw, 4.5rem)", lineHeight: 1.1 }}
          >
            {code}
          </Text>
        )}
        <Text
          ff="monospace"
          ta="center"
          c="#000000"
          style={{
            fontSize: "clamp(0.9rem, 2.2vw, 1.6rem)",
            wordBreak: "break-all",
          }}
        >
          {url}
        </Text>
        <Text size="sm" c="#666666">
          {t("share.fullscreenHint")}
        </Text>
      </Stack>
    </Modal>
  );
}
