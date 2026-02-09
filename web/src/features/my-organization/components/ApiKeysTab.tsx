import { useState, useMemo } from "react";
import {
  Stack,
  Text,
  Alert,
  Card,
  Group,
  Badge,
  Table,
  ActionIcon,
  Loader,
  Center,
  Modal,
  Select,
  Checkbox,
  TextInput,
  Button,
  Tooltip,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import {
  IconKey,
  IconInfoCircle,
  IconTrash,
  IconPlus,
  IconSearch,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { useAuth } from "@/providers/AuthProvider";
import { ActionButton, PlusIconButton } from "@/common/components/buttons";
import {
  ExpandableSearch,
  FilterSegmentedControl,
} from "@/common/components/controls";
import {
  useInstitutionApiKeys,
  useApiKeys,
  useShareApiKeyWithInstitution,
  useRemoveInstitutionApiKeyShare,
  useSetInstitutionFreeUseKey,
} from "@/api/hooks";
import type { ObjApiKeyShare } from "@/api/generated";

interface ApiKeysTabProps {
  institutionId: string;
  institutionName?: string;
  freeUseApiKeyShareId?: string;
}

export function ApiKeysTab({
  institutionId,
  institutionName,
  freeUseApiKeyShareId,
}: ApiKeysTabProps) {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const { backendUser } = useAuth();
  const [shareModalOpened, { open: openShareModal, close: closeShareModal }] =
    useDisclosure(false);
  const [selectedKeyId, setSelectedKeyId] = useState<string | null>(null);
  const [allowPublicSponsored, setAllowPublicSponsored] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [platformFilter, setPlatformFilter] = useState<string | null>(null);

  // Fetch institution API keys
  const {
    data: institutionKeys,
    isLoading,
    isError,
  } = useInstitutionApiKeys(institutionId);

  // Fetch user's own keys (for sharing)
  const { data: userKeys } = useApiKeys();

  // Mutations
  const shareApiKey = useShareApiKeyWithInstitution();
  const removeShare = useRemoveInstitutionApiKeyShare();
  const setFreeUseKey = useSetInstitutionFreeUseKey();

  // Get unique platforms for filter
  const platforms = useMemo(() => {
    if (!institutionKeys) return [];
    const unique = new Set(
      institutionKeys.map((s) => s.apiKey?.platform).filter(Boolean),
    );
    return Array.from(unique) as string[];
  }, [institutionKeys]);

  // Filter institution keys by search and platform
  const filteredKeys = useMemo(() => {
    if (!institutionKeys) return [];
    return institutionKeys.filter((share) => {
      const matchesSearch =
        !searchQuery ||
        share.apiKey?.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        share.apiKey?.userName
          ?.toLowerCase()
          .includes(searchQuery.toLowerCase());
      const matchesPlatform =
        !platformFilter || share.apiKey?.platform === platformFilter;
      return matchesSearch && matchesPlatform;
    });
  }, [institutionKeys, searchQuery, platformFilter]);

  // Filter user's own keys that aren't already shared with the institution
  const ownKeys = userKeys?.apiKeys ?? [];
  const ownShares = userKeys?.shares ?? [];
  const availableKeysToShare = ownKeys.filter((key) => {
    // Only show keys the user owns
    if (key.userId !== backendUser?.id) return false;
    // Don't show keys already shared with this institution
    const alreadyShared = institutionKeys?.some(
      (instShare) => instShare.apiKeyId === key.id,
    );
    return !alreadyShared;
  });

  const handleShareKey = async () => {
    if (!selectedKeyId) return;

    // Find the user's self-share for this API key (backend expects share ID, not key ID)
    const selfShare = ownShares.find(
      (s) =>
        s.apiKeyId === selectedKeyId &&
        s.user &&
        !s.institution &&
        !s.workshop &&
        !s.game,
    );
    if (!selfShare?.id) return;

    await shareApiKey.mutateAsync({
      shareId: selfShare.id,
      institutionId,
      allowPublicGameSponsoring: allowPublicSponsored,
    });

    closeShareModal();
    setSelectedKeyId(null);
    setAllowPublicSponsored(false);
  };

  const handleRemoveShare = async (share: ObjApiKeyShare) => {
    if (!share.id) return;
    await removeShare.mutateAsync({
      shareId: share.id,
      institutionId,
    });
  };

  if (isLoading) {
    return (
      <Center py="xl">
        <Loader />
      </Center>
    );
  }

  if (isError) {
    return (
      <Alert color="red" title={t("error")}>
        {t("myOrganization.apiKeys.loadError")}
      </Alert>
    );
  }

  const hasKeys = institutionKeys && institutionKeys.length > 0;

  return (
    <Stack gap="lg">
      {/* Info block */}
      <Alert
        icon={<IconInfoCircle size={16} />}
        title={t("myOrganization.apiKeys.aboutTitle")}
        color="cyan"
      >
        <Text size="sm">{t("myOrganization.apiKeys.aboutDescription")}</Text>
      </Alert>

      {/* Actions and filters */}
      {isMobile ? (
        <Group gap="sm" wrap="nowrap">
          <Tooltip label={t("myOrganization.apiKeys.shareKey")} withArrow>
            <PlusIconButton
              onClick={openShareModal}
              variant="filled"
              disabled={availableKeysToShare.length === 0}
              aria-label={t("myOrganization.apiKeys.shareKey")}
            />
          </Tooltip>
          <ExpandableSearch
            value={searchQuery}
            onChange={setSearchQuery}
            placeholder={t("myOrganization.apiKeys.searchPlaceholder")}
          />
          {hasKeys && platforms.length > 1 && (
            <FilterSegmentedControl
              value={platformFilter || "all"}
              onChange={(v) => setPlatformFilter(v === "all" ? null : v)}
              options={[
                {
                  value: "all",
                  label: t("myOrganization.apiKeys.allPlatforms", "All"),
                },
                ...platforms.map((p) => ({ value: p, label: p })),
              ]}
            />
          )}
        </Group>
      ) : (
        <Group justify="space-between" wrap="wrap" gap="sm">
          <ActionButton
            leftSection={<IconPlus size={16} />}
            onClick={openShareModal}
            disabled={availableKeysToShare.length === 0}
          >
            {t("myOrganization.apiKeys.shareKey")}
          </ActionButton>

          {hasKeys && (
            <Group gap="sm" wrap="wrap">
              <TextInput
                placeholder={t("myOrganization.apiKeys.searchPlaceholder")}
                leftSection={<IconSearch size={16} />}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.currentTarget.value)}
                size="sm"
                style={{ minWidth: 200 }}
              />
              <Select
                placeholder={t("myOrganization.apiKeys.filterByPlatform")}
                data={platforms.map((p) => ({ value: p, label: p }))}
                value={platformFilter}
                onChange={setPlatformFilter}
                clearable
                size="sm"
                style={{ minWidth: 150 }}
              />
            </Group>
          )}
        </Group>
      )}

      {/* Keys list */}
      {hasKeys ? (
        <Card
          shadow="sm"
          padding={isMobile ? "md" : "lg"}
          radius="md"
          withBorder
        >
          {filteredKeys.length > 0 ? (
            <Table striped={!isMobile} highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>{t("myOrganization.apiKeys.keyName")}</Table.Th>
                  <Table.Th>{t("myOrganization.apiKeys.owner")}</Table.Th>
                  <Table.Th>{t("myOrganization.apiKeys.platform")}</Table.Th>
                  {!isMobile && (
                    <Table.Th>
                      {t("myOrganization.apiKeys.publicSponsoring")}
                    </Table.Th>
                  )}
                  <Table.Th>{t("myOrganization.actions")}</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {filteredKeys.map((share) => (
                  <Table.Tr key={share.id}>
                    <Table.Td>
                      <Group gap="xs">
                        <IconKey size={16} />
                        <Text size="sm">
                          {share.apiKey?.name || t("unnamed")}
                        </Text>
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm">{share.apiKey?.userName}</Text>
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm">{share.apiKey?.platform}</Text>
                    </Table.Td>
                    {!isMobile && (
                      <Table.Td>
                        {share.allowPublicGameSponsoring ? (
                          <Badge color="red" variant="light" size="sm">
                            {t("labels.yes")}
                          </Badge>
                        ) : (
                          <Badge color="green" variant="light" size="sm">
                            {t("labels.no")}
                          </Badge>
                        )}
                      </Table.Td>
                    )}
                    <Table.Td>
                      <ActionIcon
                        color="red"
                        variant="subtle"
                        onClick={() => handleRemoveShare(share)}
                        loading={removeShare.isPending}
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          ) : (
            <Stack gap="md" align="center" py="xl">
              <IconSearch size={48} color="var(--mantine-color-dimmed)" />
              <Text c="dimmed" ta="center">
                {t("myOrganization.apiKeys.noResults")}
              </Text>
            </Stack>
          )}
        </Card>
      ) : (
        <Card
          shadow="sm"
          padding={isMobile ? "md" : "lg"}
          radius="md"
          withBorder
        >
          <Stack gap="md" align="center" py="xl">
            <IconKey size={48} color="var(--mantine-color-dimmed)" />
            <Text c="dimmed" ta="center">
              {t("myOrganization.apiKeys.noKeys")}
            </Text>
            <Badge color="cyan" variant="light">
              {institutionName || institutionId}
            </Badge>
          </Stack>
        </Card>
      )}

      {/* Free-Use Key Section */}
      <Card shadow="sm" padding={isMobile ? "md" : "lg"} radius="md" withBorder>
        <Stack gap="md">
          <Group gap="xs">
            <IconKey size={20} />
            <Text fw={600} size="sm">
              {t("myOrganization.apiKeys.freeUseKey.title")}
            </Text>
          </Group>
          <Text size="sm" c="dimmed">
            {t("myOrganization.apiKeys.freeUseKey.description")}
          </Text>

          {freeUseApiKeyShareId ? (
            (() => {
              const currentFreeUseShare = institutionKeys?.find(
                (s) => s.id === freeUseApiKeyShareId,
              );
              return (
                <Group gap="sm" wrap="wrap">
                  <Badge color="cyan" variant="light" size="lg">
                    {currentFreeUseShare?.apiKey?.name ||
                      t("myOrganization.apiKeys.freeUseKey.unknownKey")}
                    {currentFreeUseShare?.apiKey?.platform &&
                      ` (${currentFreeUseShare.apiKey.platform})`}
                  </Badge>
                  <Button
                    variant="subtle"
                    color="red"
                    size="xs"
                    onClick={() =>
                      setFreeUseKey.mutate({
                        institutionId,
                        shareId: null,
                      })
                    }
                    loading={setFreeUseKey.isPending}
                  >
                    {t("myOrganization.apiKeys.freeUseKey.remove")}
                  </Button>
                </Group>
              );
            })()
          ) : (
            <Select
              placeholder={t(
                "myOrganization.apiKeys.freeUseKey.selectPlaceholder",
              )}
              data={
                institutionKeys?.map((share) => ({
                  value: share.id || "",
                  label: `${share.apiKey?.name || t("unnamed")} (${share.apiKey?.platform})`,
                })) ?? []
              }
              onChange={(value) => {
                if (value) {
                  setFreeUseKey.mutate({
                    institutionId,
                    shareId: value,
                  });
                }
              }}
              disabled={!hasKeys || setFreeUseKey.isPending}
              clearable={false}
              size="sm"
              style={{ maxWidth: 400 }}
            />
          )}
        </Stack>
      </Card>

      {/* Share Key Modal */}
      <Modal
        opened={shareModalOpened}
        onClose={closeShareModal}
        title={t("myOrganization.apiKeys.shareKeyTitle")}
      >
        <Stack gap="md">
          <Select
            label={t("myOrganization.apiKeys.selectKey")}
            placeholder={t("myOrganization.apiKeys.selectKeyPlaceholder")}
            data={availableKeysToShare.map((key) => ({
              value: key.id || "",
              label: `${key.name || t("unnamed")} (${key.platform})`,
            }))}
            value={selectedKeyId}
            onChange={setSelectedKeyId}
          />

          <Checkbox
            label={t("myOrganization.apiKeys.allowPublicSponsoring")}
            description={t("myOrganization.apiKeys.allowPublicSponsoringDesc")}
            checked={allowPublicSponsored}
            onChange={(e) => setAllowPublicSponsored(e.currentTarget.checked)}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={closeShareModal}>
              {t("cancel")}
            </Button>
            <Button
              onClick={handleShareKey}
              loading={shareApiKey.isPending}
              disabled={!selectedKeyId}
            >
              {t("myOrganization.apiKeys.share")}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
