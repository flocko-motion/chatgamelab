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

import { useOwnGuestCode } from "@/api/hooks";
import { GuestCodeDisplay } from "@components/share";

/** The Guest's own re-login code, laid out to be photographed in one shot. */
export function GuestCodeCard() {
  const { t } = useTranslation("auth");
  const { data, isLoading, isError } = useOwnGuestCode();

  return (
    <Card shadow="sm" padding="xl" radius="md" withBorder>
      <Stack gap="md">
        <Group gap="sm">
          <ThemeIcon variant="light" size="lg" color="accent">
            <IconKey size={20} />
          </ThemeIcon>
          <Title order={3}>{t("profile.guestCode.title")}</Title>
        </Group>

        {isLoading && <Skeleton height={160} />}

        {(isError || (data && !data.token)) && (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {t("profile.guestCode.error")}
          </Alert>
        )}

        {data?.token && (
          <GuestCodeDisplay
            token={data.token}
            words={data.words}
            hint={
              data.words
                ? t("profile.guestCode.hint")
                : t("profile.guestCode.longTokenHint")
            }
          />
        )}
      </Stack>
    </Card>
  );
}
