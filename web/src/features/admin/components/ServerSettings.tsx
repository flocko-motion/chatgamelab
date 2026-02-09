import {
  Container,
  Title,
  Text,
  Stack,
  Card,
  Group,
  Badge,
  Select,
  Button,
  Alert,
  Loader,
} from "@mantine/core";
import { IconKey, IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";
import { isAdmin } from "@/common/lib/roles";
import {
  useSystemSettings,
  useSetSystemFreeUseKey,
  useApiKeys,
} from "@/api/hooks";

export function ServerSettings() {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();

  const { data: settings, isLoading: settingsLoading } = useSystemSettings();
  const { data: apiKeys, isLoading: keysLoading } = useApiKeys();
  const setFreeUseKey = useSetSystemFreeUseKey();

  if (!isAdmin(backendUser)) {
    return (
      <Container size="xl" py="xl">
        <Alert
          icon={<IconAlertCircle size={16} />}
          title={t("error")}
          color="red"
        >
          {t("serverSettings.notAuthorized")}
        </Alert>
      </Container>
    );
  }

  if (settingsLoading || keysLoading) {
    return (
      <Container size="xl" py="xl">
        <Group justify="center" py="xl">
          <Loader />
        </Group>
      </Container>
    );
  }

  // Flatten API keys from shares to get unique keys owned by the admin
  const uniqueKeys = new Map<
    string,
    { id: string; name: string; platform: string }
  >();
  apiKeys?.forEach((share) => {
    if (share.apiKey?.id && !uniqueKeys.has(share.apiKey.id)) {
      uniqueKeys.set(share.apiKey.id, {
        id: share.apiKey.id,
        name: share.apiKey.name || "",
        platform: share.apiKey.platform || "",
      });
    }
  });
  const keyOptions = Array.from(uniqueKeys.values()).map((key) => ({
    value: key.id,
    label: `${key.name} (${key.platform})`,
  }));

  const currentKeyId = settings?.freeUseApiKeyId;
  const currentKey = currentKeyId
    ? uniqueKeys.get(currentKeyId)
    : undefined;

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Title order={2}>{t("serverSettings.title")}</Title>

        {/* Free-Use Key Section */}
        <Card shadow="sm" padding="lg" radius="md" withBorder>
          <Stack gap="md">
            <Group gap="xs">
              <IconKey size={20} />
              <Text fw={600} size="sm">
                {t("serverSettings.freeUseKey.title")}
              </Text>
            </Group>
            <Text size="sm" c="dimmed">
              {t("serverSettings.freeUseKey.description")}
            </Text>

            {currentKey ? (
              <Group gap="sm" wrap="wrap">
                <Badge color="cyan" variant="light" size="lg">
                  {currentKey.name} ({currentKey.platform})
                </Badge>
                <Button
                  variant="subtle"
                  color="red"
                  size="xs"
                  onClick={() =>
                    setFreeUseKey.mutate({ apiKeyId: null })
                  }
                  loading={setFreeUseKey.isPending}
                >
                  {t("serverSettings.freeUseKey.remove")}
                </Button>
              </Group>
            ) : currentKeyId ? (
              <Group gap="sm" wrap="wrap">
                <Badge color="orange" variant="light" size="lg">
                  {t("serverSettings.freeUseKey.unknownKey")}
                </Badge>
                <Button
                  variant="subtle"
                  color="red"
                  size="xs"
                  onClick={() =>
                    setFreeUseKey.mutate({ apiKeyId: null })
                  }
                  loading={setFreeUseKey.isPending}
                >
                  {t("serverSettings.freeUseKey.remove")}
                </Button>
              </Group>
            ) : (
              <Select
                placeholder={t(
                  "serverSettings.freeUseKey.selectPlaceholder",
                )}
                data={keyOptions}
                onChange={(value) => {
                  if (value) {
                    setFreeUseKey.mutate({ apiKeyId: value });
                  }
                }}
                disabled={
                  keyOptions.length === 0 || setFreeUseKey.isPending
                }
                clearable={false}
                size="sm"
                style={{ maxWidth: 400 }}
              />
            )}

            {keyOptions.length === 0 && (
              <Text size="xs" c="dimmed">
                {t("serverSettings.freeUseKey.noKeys")}
              </Text>
            )}
          </Stack>
        </Card>
      </Stack>
    </Container>
  );
}
