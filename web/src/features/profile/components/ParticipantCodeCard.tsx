import {
  Alert,
  Card,
  Group,
  Skeleton,
  Stack,
  ThemeIcon,
  Title,
} from "@mantine/core";
import { IconAlertCircle, IconKey } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

import { useOwnParticipantCode } from "@/api/hooks";
import { ParticipantCodeDisplay } from "@components/share";

/** The participant's own re-login code, laid out to be photographed in one shot. */
export function ParticipantCodeCard() {
  const { t } = useTranslation("auth");
  const { data, isLoading, isError } = useOwnParticipantCode();

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

        {(isError || (data && !data.token)) && (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {t("profile.participantCode.error")}
          </Alert>
        )}

        {data?.token && (
          <ParticipantCodeDisplay
            token={data.token}
            words={data.words}
            hint={
              data.words
                ? t("profile.participantCode.hint")
                : t("profile.participantCode.longTokenHint")
            }
          />
        )}
      </Stack>
    </Card>
  );
}
