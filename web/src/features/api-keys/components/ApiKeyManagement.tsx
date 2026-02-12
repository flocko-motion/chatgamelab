import { useState } from "react";
import {
  Stack,
  Group,
  Card,
  Modal,
  TextInput,
  Select,
  Text,
  Alert,
  Box,
  SimpleGrid,
  Skeleton,
  ActionIcon,
  ThemeIcon,
  useMantineTheme,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import {
  IconPlus,
  IconAlertCircle,
  IconCircle,
  IconCircleCheckFilled,
  IconCircleCheck,
  IconCircleX,
  IconCircleMinus,
  IconKey,
} from "@tabler/icons-react";
import {
  ActionButton,
  TextButton,
  DangerButton,
  DeleteIconButton,
} from "@components/buttons";
import { InfoCard } from "@components/cards";
import { PageTitle } from "@components/typography";
import {
  useApiKeys,
  useCreateApiKey,
  useDeleteApiKey,
  useSetDefaultApiKey,
  usePlatforms,
  useUpdateUser,
} from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { getAiQualityTierOptions } from "@/common/lib/aiQualityTier";
import type { ObjAiPlatform } from "@/api/generated";
import env from "@/config/env";
import { ApiKeyShares } from "./ApiKeyShares";

export function ApiKeyManagement() {
  const { t } = useTranslation("common");
  const theme = useMantineTheme();
  const { backendUser, retryBackendFetch } = useAuth();
  const updateUser = useUpdateUser();
  const [
    createModalOpened,
    { open: openCreateModal, close: closeCreateModal },
  ] = useDisclosure(false);

  // Create form state
  const [createName, setCreateName] = useState("");
  const [createPlatform, setCreatePlatform] = useState("openai");
  const [createKey, setCreateKey] = useState("");
  const [createErrors, setCreateErrors] = useState<{
    name?: string;
    platform?: string;
    key?: string;
  }>({});

  const { data: apiKeysData, isLoading, error } = useApiKeys();
  const apiKeys = apiKeysData?.apiKeys ?? [];
  const allShares = apiKeysData?.shares ?? [];
  const { data: platforms, isLoading: platformsLoading } = usePlatforms();
  const createApiKey = useCreateApiKey();
  const deleteApiKey = useDeleteApiKey();
  const setDefaultApiKey = useSetDefaultApiKey();

  const [
    deleteModalOpened,
    { open: openDeleteModal, close: closeDeleteModal },
  ] = useDisclosure(false);
  const [selectedKey, setSelectedKey] = useState<{
    id: string;
    name: string;
  } | null>(null);

  const validateCreateForm = () => {
    const errors: { name?: string; platform?: string; key?: string } = {};
    if (createName.trim().length === 0)
      errors.name = t("apiKeys.errors.nameRequired");
    if (createPlatform.length === 0)
      errors.platform = t("apiKeys.errors.platformRequired");
    if (createKey.trim().length === 0)
      errors.key = t("apiKeys.errors.keyRequired");
    setCreateErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleCreateKey = async () => {
    if (!validateCreateForm()) return;
    try {
      await createApiKey.mutateAsync({
        name: createName,
        platform: createPlatform,
        key: createKey,
      });
      setCreateName("");
      setCreatePlatform("openai");
      setCreateKey("");
      setCreateErrors({});
      closeCreateModal();
    } catch {
      // Error is handled by the mutation
    }
  };

  const formatDefaultKeyName = (platformId: string) => {
    const userName = backendUser?.name || "";
    const platform = platformId.charAt(0).toUpperCase() + platformId.slice(1);
    const now = new Date();
    const dd = String(now.getDate()).padStart(2, "0");
    const mm = String(now.getMonth() + 1).padStart(2, "0");
    const yy = String(now.getFullYear()).slice(-2);
    return userName
      ? `${userName} ${platform} ${dd}.${mm}.${yy}`
      : `${platform} ${dd}.${mm}.${yy}`;
  };

  const openCreateForPlatform = (platform: ObjAiPlatform) => {
    setCreatePlatform(platform.id || "openai");
    setCreateName(formatDefaultKeyName(platform.id || "openai"));
    setCreateKey("");
    setCreateErrors({});
    openCreateModal();
  };

  const handleDeleteKey = async () => {
    if (!selectedKey?.id) return;
    try {
      await deleteApiKey.mutateAsync({
        id: selectedKey.id,
        cascade: true,
      });
      closeDeleteModal();
      setSelectedKey(null);
    } catch {
      // Error is handled by the mutation
    }
  };

  const openDelete = (keyId: string, keyName: string) => {
    setSelectedKey({ id: keyId, name: keyName });
    openDeleteModal();
  };

  if (isLoading) {
    return (
      <Stack gap="xl">
        <Skeleton height={40} width="50%" />
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
          {[1, 2].map((i) => (
            <Card key={i} shadow="sm" p="lg" radius="md" withBorder>
              <Stack gap="md">
                <Box>
                  <Skeleton height={28} width="60%" mb="xs" />
                  <Skeleton height={16} width="40%" />
                </Box>
                <Skeleton height={20} width="30%" />
              </Stack>
            </Card>
          ))}
        </SimpleGrid>
      </Stack>
    );
  }

  if (error) {
    return (
      <Alert
        icon={<IconAlertCircle size={16} />}
        title={t("errors.titles.error")}
        color="red"
      >
        {t("apiKeys.errors.loadFailed")}
      </Alert>
    );
  }

  return (
    <Stack gap="xl">
      <Box>
        <PageTitle>{t("apiKeys.title")}</PageTitle>
        <Text c="dimmed" mt="xs">
          {t("apiKeys.subtitle")}
        </Text>
      </Box>

      {/* Info Block */}
      <InfoCard title={t("apiKeys.aboutSection.title")}>
        {t("apiKeys.aboutSection.description")}
      </InfoCard>

      {/* Current Default Key */}
      {(() => {
        const defaultKey = apiKeys.find((k) => k.isDefault);
        const platformName = platforms?.find(
          (p) => p.id === defaultKey?.platform,
        )?.name;
        return (
          <Card
            shadow="sm"
            p="lg"
            radius="md"
            withBorder
            style={{ borderLeft: "4px solid var(--mantine-color-accent-5)" }}
          >
            <Group gap="md" align="center">
              <ThemeIcon variant="light" color="accent" size="lg" radius="md">
                <IconKey size={20} />
              </ThemeIcon>
              {defaultKey ? (
                <Box style={{ flex: 1 }}>
                  <Text size="xs" c="dimmed" tt="uppercase" fw={600}>
                    {t("apiKeys.currentDefault.label")}
                  </Text>
                  <Text size="sm" fw={700} mt={2}>
                    {defaultKey.name || t("apiKeys.unnamed")}
                  </Text>
                  <Group gap="md" mt={4}>
                    <Text size="xs" c="dimmed">
                      {t("apiKeys.platform")}:{" "}
                      {platformName || defaultKey.platform}
                    </Text>
                    <Group gap={4} align="center">
                      {defaultKey.lastUsageSuccess === true && (
                        <>
                          <IconCircleCheck
                            size={14}
                            color="var(--mantine-color-green-6)"
                          />
                          <Text size="xs" c="green.7" fw={600}>
                            {t("apiKeys.status.working")}
                          </Text>
                        </>
                      )}
                      {defaultKey.lastUsageSuccess === false && (
                        <>
                          <IconCircleX
                            size={14}
                            color="var(--mantine-color-red-6)"
                          />
                          <Text size="xs" c="red.7" fw={600}>
                            {t("apiKeys.status.failed")}
                          </Text>
                        </>
                      )}
                      {defaultKey.lastUsageSuccess == null && (
                        <>
                          <IconCircleMinus
                            size={14}
                            color="var(--mantine-color-gray-5)"
                          />
                          <Text size="xs" c="dimmed" fw={600}>
                            {t("apiKeys.status.unknown")}
                          </Text>
                        </>
                      )}
                    </Group>
                  </Group>
                </Box>
              ) : (
                <Box style={{ flex: 1 }}>
                  <Text size="xs" c="dimmed" tt="uppercase" fw={600}>
                    {t("apiKeys.currentDefault.label")}
                  </Text>
                  <Text size="sm" c="dimmed" mt={2}>
                    {t("apiKeys.currentDefault.none")}
                  </Text>
                </Box>
              )}
            </Group>
          </Card>
        );
      })()}

      {/* User AI Quality Tier */}
      <Card shadow="sm" p="lg" radius="md" withBorder>
        <Stack gap="sm">
          <Text size="xs" c="dimmed" tt="uppercase" fw={600}>
            {t("aiQualityTier.label")}
          </Text>
          <Text size="sm" c="dimmed">
            {t("apiKeys.aiQualityTier.description")}
          </Text>
          <Select
            data={getAiQualityTierOptions(t, { includeEmpty: true })}
            value={backendUser?.aiQualityTier || ""}
            onChange={(value) => {
              if (!backendUser?.id) return;
              updateUser.mutate(
                {
                  id: backendUser.id,
                  request: { aiQualityTier: value || "" },
                },
                { onSuccess: () => retryBackendFetch() },
              );
            }}
            disabled={updateUser.isPending}
            size="sm"
            style={{ maxWidth: 300 }}
          />
        </Stack>
      </Card>

      {/* Platform Cards */}
      {platformsLoading ? (
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
          {[1, 2].map((i) => (
            <Card key={i} shadow="sm" p="lg" radius="md" withBorder>
              <Stack gap="md">
                <Box>
                  <Skeleton height={28} width="60%" mb="xs" />
                  <Skeleton height={16} width="40%" />
                </Box>
                <Skeleton height={20} width="30%" />
              </Stack>
            </Card>
          ))}
        </SimpleGrid>
      ) : (
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
          {platforms
            ?.filter(
              (p) =>
                (p.id !== "mock" || env.DEV) &&
                (p.supportsApiKey ||
                  apiKeys.some((k) => k.platform === p.id) ||
                  false),
            )
            .map((platform) => {
              const platformKeys =
                apiKeys.filter((k) => k.platform === platform.id) || [];
              return (
                <Card
                  key={platform.id}
                  shadow="sm"
                  p="lg"
                  radius="md"
                  withBorder
                  style={{
                    borderTop: platform.supportsApiKey
                      ? "3px solid var(--mantine-color-accent-5)"
                      : "3px solid var(--mantine-color-red-5)",
                  }}
                >
                  <Stack gap="md">
                    <Group justify="space-between" align="flex-start">
                      <Box>
                        <Group gap="xs" align="center">
                          <Text size="lg" fw={700}>
                            {platform.name}
                          </Text>
                          {!platform.supportsApiKey && (
                            <Text size="xs" c="red" fw={600}>
                              ({t("apiKeys.unsupportedPlatform.badge")})
                            </Text>
                          )}
                        </Group>
                        <Text size="sm" c="dimmed">
                          {platformKeys.length}{" "}
                          {platformKeys.length === 1
                            ? t("apiKeys.key")
                            : t("apiKeys.keys")}
                        </Text>
                        {!platform.supportsApiKey && (
                          <Text size="xs" c="red" mt={4}>
                            {t("apiKeys.unsupportedPlatform.hint")}
                          </Text>
                        )}
                      </Box>
                    </Group>

                    {platformKeys.length > 0 && (
                      <Stack
                        gap={0}
                        style={{
                          backgroundColor: theme.colors.gray[0],
                          borderRadius: theme.radius.sm,
                          border: `1px solid ${theme.colors.gray[2]}`,
                        }}
                      >
                        {platformKeys.map((key, index) => {
                          const isDefault = key.isDefault;
                          const usageStatus = key.lastUsageSuccess;
                          const keyShares = allShares.filter(
                            (s) => s.apiKeyId === key.id,
                          );
                          return (
                            <Box
                              key={key.id}
                              style={{
                                borderTop:
                                  index === 0
                                    ? "none"
                                    : `1px solid ${theme.colors.gray[2]}`,
                                ...(isDefault
                                  ? {
                                    backgroundColor:
                                      "var(--mantine-color-accent-0)",
                                  }
                                  : {}),
                              }}
                            >
                              <Group
                                justify="space-between"
                                align="center"
                                px="sm"
                                py="xs"
                                wrap="nowrap"
                              >
                                <Box
                                  style={{
                                    flexShrink: 0,
                                    display: "flex",
                                    alignItems: "center",
                                    justifyContent: "center",
                                    width: 26,
                                    height: 26,
                                  }}
                                >
                                  {isDefault ? (
                                    <IconCircleCheckFilled
                                      size={26}
                                      color="var(--mantine-color-accent-5)"
                                    />
                                  ) : (
                                    <ActionIcon
                                      variant="transparent"
                                      color="gray"
                                      size="md"
                                      onClick={() =>
                                        key.id &&
                                        setDefaultApiKey.mutate({
                                          id: key.id,
                                        })
                                      }
                                      loading={setDefaultApiKey.isPending}
                                      title={t("apiKeys.setDefault")}
                                    >
                                      <IconCircle size={22} />
                                    </ActionIcon>
                                  )}
                                </Box>
                                <Box style={{ flex: 1, minWidth: 0 }}>
                                  <Text
                                    size="sm"
                                    fw={isDefault ? 700 : 600}
                                    truncate
                                  >
                                    {key.name || t("apiKeys.unnamed")}
                                  </Text>
                                  <Text size="xs" c="dimmed" mt={2}>
                                    {t("apiKeys.addedOn")}:{" "}
                                    {key.meta?.createdAt
                                      ? new Date(
                                        key.meta.createdAt,
                                      ).toLocaleDateString()
                                      : "-"}
                                  </Text>
                                </Box>
                                <Group
                                  gap={4}
                                  align="center"
                                  wrap="nowrap"
                                  style={{
                                    flexShrink: 0,
                                    minWidth: 120,
                                    justifyContent: "flex-end",
                                  }}
                                >
                                  {usageStatus === true && (
                                    <Group gap={4} align="center" wrap="nowrap">
                                      <IconCircleCheck
                                        size={14}
                                        color="var(--mantine-color-green-6)"
                                        style={{ flexShrink: 0 }}
                                      />
                                      <Text
                                        size="xs"
                                        c="green.7"
                                        fw={600}
                                        style={{ whiteSpace: "nowrap" }}
                                      >
                                        {t("apiKeys.status.working")}
                                      </Text>
                                    </Group>
                                  )}
                                  {usageStatus === false && (
                                    <Group gap={4} align="center" wrap="nowrap">
                                      <IconCircleX
                                        size={14}
                                        color="var(--mantine-color-red-6)"
                                        style={{ flexShrink: 0 }}
                                      />
                                      <Text
                                        size="xs"
                                        c="red.7"
                                        fw={600}
                                        style={{ whiteSpace: "nowrap" }}
                                      >
                                        {t("apiKeys.status.failed")}
                                      </Text>
                                    </Group>
                                  )}
                                  {usageStatus == null && (
                                    <Group gap={4} align="center" wrap="nowrap">
                                      <IconCircleMinus
                                        size={14}
                                        color="var(--mantine-color-gray-5)"
                                        style={{ flexShrink: 0 }}
                                      />
                                      <Text
                                        size="xs"
                                        c="dimmed"
                                        fw={600}
                                        style={{ whiteSpace: "nowrap" }}
                                      >
                                        {t("apiKeys.status.unknown")}
                                      </Text>
                                    </Group>
                                  )}
                                  <DeleteIconButton
                                    onClick={() =>
                                      openDelete(
                                        key.id || "",
                                        key.name || t("apiKeys.unnamed"),
                                      )
                                    }
                                    aria-label={t("delete")}
                                  />
                                </Group>
                              </Group>
                              <ApiKeyShares shares={keyShares} />
                            </Box>
                          );
                        })}
                      </Stack>
                    )}

                    {platform.supportsApiKey && (
                      <TextButton
                        size="xs"
                        leftSection={<IconPlus size={14} />}
                        onClick={() => openCreateForPlatform(platform)}
                      >
                        {t("apiKeys.addKey")}
                      </TextButton>
                    )}
                  </Stack>
                </Card>
              );
            })}
        </SimpleGrid>
      )}

      {/* Create Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title={t("apiKeys.createModal.title")}
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label={t("apiKeys.createModal.nameLabel")}
            placeholder={`My ${createPlatform.charAt(0).toUpperCase() + createPlatform.slice(1)}`}
            value={createName}
            onChange={(e) => setCreateName(e.currentTarget.value)}
            error={createErrors.name}
          />
          <Select
            label={t("apiKeys.createModal.platformLabel")}
            placeholder={t("apiKeys.createModal.platformPlaceholder")}
            data={
              platforms?.map((p) => ({
                value: p.id || "",
                label: p.name || "",
              })) || []
            }
            required
            value={createPlatform}
            onChange={(value) => setCreatePlatform(value || "openai")}
            error={createErrors.platform}
            disabled
          />
          <TextInput
            label={t("apiKeys.createModal.keyLabel")}
            placeholder={t("apiKeys.createModal.keyPlaceholder")}
            description={t("apiKeys.createModal.keyDescription")}
            required
            type="password"
            value={createKey}
            onChange={(e) => setCreateKey(e.currentTarget.value)}
            error={createErrors.key}
          />
          <Group justify="flex-end" mt="md">
            <TextButton onClick={closeCreateModal}>{t("cancel")}</TextButton>
            <ActionButton
              size="md"
              onClick={handleCreateKey}
              loading={createApiKey.isPending}
            >
              {t("apiKeys.createModal.submit")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        title={t("apiKeys.deleteModal.title")}
        size="sm"
      >
        <Stack gap="md">
          <Text>
            {t("apiKeys.deleteModal.message", {
              name: selectedKey?.name || t("apiKeys.unnamed"),
            })}
          </Text>
          <Alert
            icon={<IconAlertCircle size={16} />}
            color="red"
            variant="light"
          >
            {t("apiKeys.deleteModal.warning")}
          </Alert>
          <Group justify="flex-end" mt="md">
            <TextButton onClick={closeDeleteModal}>{t("cancel")}</TextButton>
            <DangerButton
              onClick={handleDeleteKey}
              loading={deleteApiKey.isPending}
            >
              {t("delete")}
            </DangerButton>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
