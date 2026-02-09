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
import { IconKey, IconAlertCircle, IconSparkles } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";
import { isAdmin } from "@/common/lib/roles";
import {
  useSystemSettings,
  useUpdateSystemSettings,
  useSetSystemFreeUseKey,
  useApiKeys,
} from "@/api/hooks";
import { getAiQualityTierOptions } from "@/common/lib/aiQualityTier";

export function ServerSettings() {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();

  const { data: settings, isLoading: settingsLoading } = useSystemSettings();
  const { data: apiKeys, isLoading: keysLoading } = useApiKeys();
  const setFreeUseKey = useSetSystemFreeUseKey();
  const updateSettings = useUpdateSystemSettings();

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

  // Get unique keys owned by the admin
  const keys = apiKeys?.apiKeys ?? [];
  const keyOptions = keys.map((key) => ({
    value: key.id!,
    label: `${key.name || ""} (${key.platform || ""})`,
  }));
  const uniqueKeys = new Map(
    keys.map((key) => [
      key.id!,
      { id: key.id!, name: key.name || "", platform: key.platform || "" },
    ]),
  );

  const currentKeyId = settings?.freeUseApiKeyId;
  const currentKey = currentKeyId ? uniqueKeys.get(currentKeyId) : undefined;

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Title order={2}>{t("serverSettings.title")}</Title>

        {/* Default AI Quality Tier */}
        <Card shadow="sm" padding="lg" radius="md" withBorder>
          <Stack gap="md">
            <Group gap="xs">
              <IconSparkles size={20} />
              <Text fw={600} size="sm">
                {t("serverSettings.defaultTier.title")}
              </Text>
            </Group>
            <Text size="sm" c="dimmed">
              {t("serverSettings.defaultTier.description")}
            </Text>
            <Select
              data={getAiQualityTierOptions(t)}
              value={settings?.defaultAiQualityTier || "medium"}
              onChange={(value) => {
                if (value) {
                  updateSettings.mutate({ defaultAiQualityTier: value });
                }
              }}
              disabled={updateSettings.isPending}
              size="sm"
              style={{ maxWidth: 300 }}
            />
          </Stack>
        </Card>

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
                  onClick={() => setFreeUseKey.mutate({ apiKeyId: null })}
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
                  onClick={() => setFreeUseKey.mutate({ apiKeyId: null })}
                  loading={setFreeUseKey.isPending}
                >
                  {t("serverSettings.freeUseKey.remove")}
                </Button>
              </Group>
            ) : (
              <Select
                placeholder={t("serverSettings.freeUseKey.selectPlaceholder")}
                data={keyOptions}
                onChange={(value) => {
                  if (value) {
                    setFreeUseKey.mutate({ apiKeyId: value });
                  }
                }}
                disabled={keyOptions.length === 0 || setFreeUseKey.isPending}
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

            <Text size="sm" c="dimmed">
              {t("serverSettings.freeUseTier.description")}
            </Text>
            <Select
              label={t("serverSettings.freeUseTier.title")}
              data={getAiQualityTierOptions(t, { includeEmpty: true })}
              value={settings?.freeUseAiQualityTier || ""}
              onChange={(value) => {
                updateSettings.mutate({
                  freeUseAiQualityTier: value || "",
                });
              }}
              disabled={updateSettings.isPending}
              size="sm"
              style={{ maxWidth: 300 }}
            />
          </Stack>
        </Card>
      </Stack>
    </Container>
  );
}
