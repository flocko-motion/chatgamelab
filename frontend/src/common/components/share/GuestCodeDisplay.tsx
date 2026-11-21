import {
  ActionIcon,
  CopyButton,
  Group,
  Stack,
  Text,
  Tooltip,
} from "@mantine/core";
import { IconCheck, IconCopy } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

import { QrCode } from "./QrCode";
import { buildShareUrl } from "@/common/lib/url";
import { guestShareLinkPath } from "@/common/lib/wordToken";

interface GuestCodeDisplayProps {
  /** The full token of a Guest in a workshop, with its prefix. */
  token: string;
  /** Its word code, or null/undefined for the long tokens issued before word tokens. */
  words?: string | null;
  /** One line under the code, worded by the caller for its situation. */
  hint?: string;
}

/**
 * The re-login code of a Guest in a workshop, laid out to be photographed in one shot: QR code,
 * the words beside it, and the link. Takes the code as a prop and fetches nothing,
 * so it also works where there is no usable session - see InactiveWorkshopMessage.
 */
export function GuestCodeDisplay({
  token,
  words,
  hint,
}: GuestCodeDisplayProps) {
  const { t } = useTranslation("common");
  const url = buildShareUrl(guestShareLinkPath(token));

  return (
    <Group gap="xl" align="center" wrap="wrap">
      <QrCode value={url} size={180} />
      <Stack gap="sm" style={{ flex: 1, minWidth: 220 }}>
        {words && (
          <Text
            ff="monospace"
            fw={700}
            style={{
              fontSize: "clamp(1.3rem, 4vw, 2rem)",
              // break-word, not break-all: a break inside a word makes the code
              // unreadable in a photo.
              wordBreak: "break-word",
            }}
          >
            {words.split("-").join(" ")}
          </Text>
        )}
        <Group gap="xs" wrap="nowrap">
          <Text
            size="sm"
            ff="monospace"
            c="dimmed"
            style={{ wordBreak: "break-all" }}
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
        {hint && <Text size="sm">{hint}</Text>}
      </Stack>
    </Group>
  );
}
