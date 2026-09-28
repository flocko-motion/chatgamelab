import {
  ActionIcon,
  Alert,
  Card,
  CopyButton,
  Group,
  Skeleton,
  Stack,
  Text,
  ThemeIcon,
  Title,
  Tooltip,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconCheck,
  IconCopy,
  IconKey,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

import { useOwnParticipantCode } from "@/api/hooks";
import { QrCode } from "@components/share";
import { buildShareUrl } from "@/common/lib/url";
import { participantShareLinkPath } from "@/common/lib/wordToken";

/** The participant's own re-login code, laid out to be photographed in one shot. */
export function ParticipantCodeCard() {
  const { t } = useTranslation("auth");
  const { t: tCommon } = useTranslation("common");
  const { data, isLoading, isError } = useOwnParticipantCode();

  const url = data?.token
    ? buildShareUrl(participantShareLinkPath(data.token))
    : null;

  return (
    <Card shadow="sm" padding="xl" radius="md" withBorder>
      <Stack gap="md">
        <Group gap="sm">
          <ThemeIcon variant="light" size="lg" color="accent">
            <IconKey size={20} />
          </ThemeIcon>
          <Title order={3}>{t("profile.participantCode.title")}</Title>
        </Group>

        {isLoading && <Skeleton height={160} />}

        {(isError || (data && !url)) && (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {t("profile.participantCode.error")}
          </Alert>
        )}

        {url && (
          <Group gap="xl" align="center" wrap="wrap">
            <QrCode value={url} size={180} />
            <Stack gap="sm" style={{ flex: 1, minWidth: 220 }}>
              {data?.words && (
                <Text
                  ff="monospace"
                  fw={700}
                  style={{
                    fontSize: "clamp(1.3rem, 4vw, 2rem)",
                    wordBreak: "break-word",
                  }}
                >
                  {data.words.split("-").join(" ")}
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
                    <Tooltip
                      label={
                        copied
                          ? tCommon("share.copied")
                          : tCommon("share.copyLink")
                      }
                    >
                      <ActionIcon
                        variant="subtle"
                        color={copied ? "green" : "gray"}
                        onClick={copy}
                        aria-label={tCommon("share.copyLink")}
                      >
                        {copied ? (
                          <IconCheck size={18} />
                        ) : (
                          <IconCopy size={18} />
                        )}
                      </ActionIcon>
                    </Tooltip>
                  )}
                </CopyButton>
              </Group>
              <Text size="sm">
                {data?.words
                  ? t("profile.participantCode.hint")
                  : t("profile.participantCode.longTokenHint")}
              </Text>
            </Stack>
          </Group>
        )}
      </Stack>
    </Card>
  );
}
