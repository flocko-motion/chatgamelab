import { Alert, Badge, Card, Code, Group, Modal, Select, Stack, TextInput, ThemeIcon, Tooltip } from "@mantine/core";
import { useTranslation } from "react-i18next";
import {
  IconAlertCircle,
  IconCircle,
  IconCircleCheckFilled,
  IconKey,
  IconPlus,
  IconRefresh,
} from "@tabler/icons-react";
import {
  ActionButton,
  DangerButton,
  DeleteIconButton,
  GenericIconButton,
  TextButton,
} from "@components/buttons";
import { InfoCard } from "@components/cards";
import { ListTable, type ListColumn } from "@components/ListTable";
import { BodyText, HelperText, Label, PageTitle } from "@components/typography";
import { HelpLink } from "@components/HelpLink";
import { HELP_LINKS } from "@/config/helpLinks";
import type { OwnApiKey, OwnKeyListSort } from "@/api/hooks";
import {
  getAiQualityTierLabel,
  getAiQualityTierOptions,
  normalizeAiQualityTier,
} from "@/common/lib/aiQualityTier";
import { translateErrorCode } from "@/common/lib/errorHelpers";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { keyShareGroups } from "../lib/keyShareGroups";
import { useApiKeyManagement } from "../hooks/useApiKeyManagement";
import { ApiKeyShares } from "./ApiKeyShares";
import { KeyStatus } from "./KeyStatus";

/**
 * The caller's API keys: the default key and quality tier, and every key in
 * one table, whatever its platform, with what it is shared as.
 */
export function ApiKeyManagement() {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const state = useApiKeyManagement(t);
  const { list, defaultKey } = state;

  const keyName = (item: OwnApiKey) => item.key.name || t("apiKeys.unnamed");
  // The key's last characters, so two keys of one name are told apart.
  const keyTail = (shortened: string) =>
    shortened ? (
      <Tooltip label={t("apiKeys.keyEndsWith")} withArrow>
        <Code>{shortened}</Code>
      </Tooltip>
    ) : null;
  const addedOn = (item: OwnApiKey) =>
    item.key.meta?.createdAt ? new Date(item.key.meta.createdAt).toLocaleString() : "-";
  // Why the key failed: the engine's problem, said for a key's owner.
  const lastError = (item: OwnApiKey) =>
    item.key.lastUsageSuccess === false && item.key.lastErrorCode
      ? keyProblem(item.key.lastErrorCode)
      : null;
  const keyProblem = (code: string) =>
    t(`apiKeys.status.reasons.${code}`, { defaultValue: "" }) ||
    translateErrorCode(code)?.message ||
    code;
  const status = (item: OwnApiKey) => (
    <KeyStatus
      success={item.key.lastUsageSuccess}
      checking={state.isChecking(item)}
      checkedAt={state.checkedAt(item)}
    />
  );
  const sharedCount = (item: OwnApiKey) => keyShareGroups(item.shares).total;

  const defaultToggle = (item: OwnApiKey) =>
    item.key.isDefault ? (
      <Tooltip label={t("apiKeys.default")} withArrow>
        <IconCircleCheckFilled size={22} color="var(--mantine-color-accent-5)" />
      </Tooltip>
    ) : (
      <GenericIconButton
        icon={<IconCircle size={20} />}
        onClick={() => state.makeDefault(item)}
        disabled={state.makingDefault || !item.selfShare}
        aria-label={t("apiKeys.setDefault")}
      />
    );

  const platformCell = (item: OwnApiKey) => {
    const platform = state.platformOf(item.key.platform);
    return (
      <Group gap={6} wrap="nowrap">
        <BodyText size="sm">{platform?.name ?? item.key.platform}</BodyText>
        {platform && !platform.supported && (
          <Tooltip label={t("apiKeys.unsupportedPlatform.hint")} multiline w={260} withArrow>
            <Badge size="xs" color="red" variant="light">
              {t("apiKeys.unsupportedPlatform.badge")}
            </Badge>
          </Tooltip>
        )}
      </Group>
    );
  };

  // The key checked again now: one small billed call to its provider.
  const checkButton = (item: OwnApiKey) =>
    item.selfShare ? (
      <Tooltip label={t("apiKeys.check")} withArrow>
        <GenericIconButton
          icon={<IconRefresh size={18} />}
          onClick={() => state.checkKey(item)}
          disabled={state.isChecking(item)}
          aria-label={t("apiKeys.check")}
        />
      </Tooltip>
    ) : null;

  const actions = (item: OwnApiKey) => (
    <Group gap={2} wrap="nowrap" justify="flex-end">
      {checkButton(item)}
      <DeleteIconButton onClick={() => state.deleteModal.open(item)} aria-label={t("delete")} />
    </Group>
  );

  const columns: ListColumn<OwnApiKey, OwnKeyListSort>[] = [
    { key: "default", header: t("apiKeys.default"), render: defaultToggle, width: 80, interactive: true },
    {
      key: "name",
      header: t("name"),
      sortBy: "name",
      render: (item) => (
        <Stack gap={2}>
          <Group gap={6} wrap="nowrap">
            <BodyText size="sm" fw={item.key.isDefault ? 700 : 500}>
              {keyName(item)}
            </BodyText>
            {keyTail(item.key.keyShortened)}
          </Group>
          {lastError(item) && <HelperText c="red.6">{lastError(item)}</HelperText>}
        </Stack>
      ),
    },
    { key: "platform", header: t("apiKeys.platform"), sortBy: "platform", render: platformCell },
    {
      key: "status",
      header: t("apiKeys.statusColumn"),
      sortBy: "status",
      render: status,
    },
    {
      key: "shares",
      header: t("apiKeys.shares.toggle"),
      visibleFrom: "md",
      render: (item) =>
        sharedCount(item) > 0 ? (
          <Badge size="sm" variant="light" color="gray">
            {sharedCount(item)}
          </Badge>
        ) : null,
    },
    {
      key: "added",
      header: t("apiKeys.addedOn"),
      sortBy: "createdAt",
      visibleFrom: "lg",
      render: (item) => <HelperText>{addedOn(item)}</HelperText>,
    },
    { key: "actions", header: "", render: actions, width: 100, interactive: true },
  ];

  const renderCard = (item: OwnApiKey) => (
    <Card withBorder radius="md" padding="sm">
      <Group justify="space-between" wrap="nowrap" align="flex-start">
        <Group gap="sm" wrap="nowrap" align="flex-start" style={{ minWidth: 0 }}>
          {defaultToggle(item)}
          <Stack gap={2} style={{ minWidth: 0 }}>
            <Group gap={6} wrap="nowrap">
              <BodyText size="sm" fw={item.key.isDefault ? 700 : 500}>
                {keyName(item)}
              </BodyText>
              {keyTail(item.key.keyShortened)}
            </Group>
            {platformCell(item)}
            {status(item)}
            {lastError(item) && <HelperText c="red.6">{lastError(item)}</HelperText>}
            <HelperText>
              {t("apiKeys.addedOn")}: {addedOn(item)}
            </HelperText>
          </Stack>
        </Group>
        {actions(item)}
      </Group>
    </Card>
  );

  const addButton = state.canCreate &&
    (isMobile ? (
      <GenericIconButton
        icon={<IconPlus size={16} />}
        onClick={state.openCreate}
        variant="filled"
        color="accent"
        aria-label={t("apiKeys.addKey")}
      />
    ) : (
      <ActionButton onClick={state.openCreate} leftSection={<IconPlus size={16} />}>
        {t("apiKeys.addKey")}
      </ActionButton>
    ));

  const { createModal, deleteModal, qualityTier } = state;

  return (
    <Stack gap="xl">
      <Stack gap="xs">
        <Group gap="sm" align="center">
          <PageTitle>{t("apiKeys.title")}</PageTitle>
          <HelpLink href={HELP_LINKS.API_KEY_FAQ} />
        </Group>
        <HelperText>{t("apiKeys.subtitle")}</HelperText>
      </Stack>

      <InfoCard title={t("apiKeys.aboutSection.title")}>
        {t("apiKeys.aboutSection.description")}
      </InfoCard>

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
          <Stack gap={2} style={{ flex: 1 }}>
            <Label uppercase>{t("apiKeys.currentDefault.label")}</Label>
            {defaultKey ? (
              <>
                <Group gap={6} wrap="nowrap">
                  <BodyText size="sm" fw={700}>
                    {defaultKey.name || t("apiKeys.unnamed")}
                  </BodyText>
                  {keyTail(defaultKey.keyShortened)}
                </Group>
                <Group gap="md">
                  <HelperText>
                    {t("apiKeys.platform")}: {state.defaultKeyPlatformName}
                  </HelperText>
                  <KeyStatus success={defaultKey.lastUsageSuccess} />
                </Group>
                {defaultKey.lastUsageSuccess === false && defaultKey.lastErrorCode && (
                  <HelperText c="red.6">
                    {keyProblem(defaultKey.lastErrorCode)}
                  </HelperText>
                )}
              </>
            ) : (
              <HelperText>{t("apiKeys.currentDefault.none")}</HelperText>
            )}
          </Stack>
        </Group>
      </Card>

      <Card shadow="sm" p="lg" radius="md" withBorder>
        <Stack gap="sm">
          <Label uppercase>{t("aiQualityTier.label")}</Label>
          <HelperText>{t("apiKeys.aiQualityTier.description")}</HelperText>
          <Select
            data={getAiQualityTierOptions(t, { includeEmpty: true, platform: qualityTier.platform })}
            value={normalizeAiQualityTier(qualityTier.value)}
            onChange={qualityTier.set}
            disabled={qualityTier.saving}
            size="sm"
            style={{ maxWidth: 300 }}
          />
          <HelperText>{t("aiQualityTier.newSessionsOnly")}</HelperText>
        </Stack>
      </Card>

      <ListTable
        list={list}
        columns={columns}
        getRowKey={(item) => item.key.id}
        renderCard={renderCard}
        getRowAccent={(item) => (item.key.isDefault ? "accent" : undefined)}
        renderExpanded={(item) =>
          sharedCount(item) > 0 ? <ApiKeyShares shares={item.shares} /> : null
        }
        actions={addButton}
        empty={{ title: state.narrowed ? t("apiKeys.noResults") : t("apiKeys.empty") }}
        errorMessage={t("apiKeys.errors.loadFailed")}
      />

      <Modal
        opened={createModal.opened}
        onClose={createModal.close}
        title={t("apiKeys.createModal.title")}
        size="md"
      >
        <Stack gap="md">
          <Select
            label={t("apiKeys.createModal.platformLabel")}
            placeholder={t("apiKeys.createModal.platformPlaceholder")}
            data={createModal.platforms.map((p) => ({ value: p.id, label: p.name }))}
            required
            value={createModal.platform || null}
            onChange={(value) => value && createModal.setPlatform(value)}
            error={createModal.errors.platform}
            allowDeselect={false}
          />
          {createModal.tiers.length > 0 && (
            <Stack gap={0}>
              {createModal.tiers.map((tier) => (
                <HelperText key={tier.id}>
                  {getAiQualityTierLabel(t, tier.id ?? "")}: {tier.models?.story?.name}
                </HelperText>
              ))}
            </Stack>
          )}
          <TextInput
            label={t("apiKeys.createModal.nameLabel")}
            placeholder={t("apiKeys.createModal.namePlaceholder")}
            value={createModal.name}
            onChange={(e) => createModal.setName(e.currentTarget.value)}
            error={createModal.errors.name}
          />
          <TextInput
            label={t("apiKeys.createModal.keyLabel")}
            placeholder={t("apiKeys.createModal.keyPlaceholder")}
            description={t("apiKeys.createModal.keyDescription")}
            required
            type="password"
            value={createModal.key}
            onChange={(e) => createModal.setKey(e.currentTarget.value)}
            error={createModal.errors.key}
          />
          <Group justify="flex-end" mt="md">
            <TextButton onClick={createModal.close}>{t("cancel")}</TextButton>
            <ActionButton size="md" onClick={createModal.submit} loading={createModal.loading}>
              {t("apiKeys.createModal.submit")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      <Modal
        opened={deleteModal.opened}
        onClose={deleteModal.close}
        title={t("apiKeys.deleteModal.title")}
        size="sm"
      >
        <Stack gap="md">
          <BodyText>
            {t("apiKeys.deleteModal.message", {
              name: deleteModal.item ? keyName(deleteModal.item) : t("apiKeys.unnamed"),
            })}
          </BodyText>
          <Alert icon={<IconAlertCircle size={16} />} color="red" variant="light">
            {t("apiKeys.deleteModal.warning")}
          </Alert>
          <Group justify="flex-end" mt="md">
            <TextButton onClick={deleteModal.close}>{t("cancel")}</TextButton>
            <DangerButton onClick={deleteModal.confirm} loading={deleteModal.loading}>
              {t("delete")}
            </DangerButton>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
