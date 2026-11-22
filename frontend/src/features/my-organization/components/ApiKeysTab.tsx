import { Alert, Badge, Card, Group, Modal, Select, Stack, Tooltip } from "@mantine/core";
import { IconInfoCircle, IconKey, IconPlus } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { OrganisationKeyListSort } from "@/api/hooks";
import type { KeyShareView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import {
  ActionButton,
  DangerButton,
  DeleteIconButton,
  PlusIconButton,
  TextButton,
} from "@components/buttons";
import { BodyText, HelperText, Label } from "@components/typography";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { ORG_KEY_COLUMNS, type OrgKeyColumnKey } from "../lib/orgKeyListPresets";
import { useOrgApiKeysTab, type OrgApiKeysTabState } from "../hooks/useOrgApiKeysTab";
import { AutoShareConfirmModal } from "./AutoShareConfirmModal";
import { OrgKeyDetails } from "./OrgKeyDetails";

interface ApiKeysTabProps {
  institutionId: string;
  institutionName?: string;
  freeUseApiKeyShareId?: string;
}

type OrgKeyColumn = ListColumn<KeyShareView, OrganisationKeyListSort>;

/**
 * The keys given to the organisation: search, a filter per platform, headers
 * that sort; a key opens to show its game share links. Below, the free-use key.
 */
export function ApiKeysTab({
  institutionId,
  institutionName,
  freeUseApiKeyShareId,
}: ApiKeysTabProps) {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const state = useOrgApiKeysTab({ institutionId, freeUseApiKeyShareId });
  const { list, shareModal, freeUse, autoShareModal } = state;
  const allColumns = useOrgKeyColumns(state);
  const columns = ORG_KEY_COLUMNS.map((key) => allColumns[key]);

  // UI state: the cards opened on a phone (a table row opens by itself).
  const renderCard = (share: KeyShareView) => {
    return (
      <Card withBorder radius="sm" padding="sm">
        <Group gap="xs" justify="space-between" wrap="nowrap">
          <Group gap="xs" wrap="nowrap" style={{ minWidth: 0, flex: 1 }}>
            <IconKey size={16} style={{ flexShrink: 0 }} />
            <Stack gap={0} style={{ minWidth: 0 }}>
              <BodyText size="sm" fw={500}>
                {share.apiKey?.name || t("unnamed")}
              </BodyText>
              <Group gap={4}>
                <HelperText>{share.apiKey?.userName}</HelperText>
                {share.apiKey?.platform && (
                  <Badge size="xs" variant="light" color="gray">
                    {state.platformName(share.apiKey.platform)}
                  </Badge>
                )}
              </Group>
            </Stack>
          </Group>
          <RemoveKeyButton share={share} state={state} size="sm" />
        </Group>
      </Card>
    );
  };

  const shareAction = isMobile ? (
    <Tooltip label={t("myOrganization.apiKeys.shareKey")} withArrow>
      <PlusIconButton
        onClick={shareModal.open}
        variant="filled"
        disabled={!state.canShare}
        aria-label={t("myOrganization.apiKeys.shareKey")}
      />
    </Tooltip>
  ) : (
    <ActionButton
      leftSection={<IconPlus size={16} />}
      onClick={shareModal.open}
      disabled={!state.canShare}
    >
      {t("myOrganization.apiKeys.shareKey")}
    </ActionButton>
  );

  return (
    <Stack gap="lg">
      {/* Info block */}
      <Alert
        icon={<IconInfoCircle size={16} />}
        title={t("myOrganization.apiKeys.aboutTitle")}
        color="cyan"
      >
        <BodyText size="sm">{t("myOrganization.apiKeys.aboutDescription")}</BodyText>
      </Alert>

      <ListTable
        list={list}
        columns={columns}
        getRowKey={(share) => share.id}
        renderCard={renderCard}
        renderExpanded={(share) =>
          <OrgKeyDetails shareId={share.id} />
        }
        actions={shareAction}
        empty={
          state.narrowed
            ? { title: t("myOrganization.apiKeys.noResults") }
            : {
                title: t("myOrganization.apiKeys.noKeys"),
                description: institutionName || institutionId,
                icon: <IconKey size={48} color="var(--mantine-color-dimmed)" />,
              }
        }
        errorMessage={t("myOrganization.apiKeys.loadError")}
      />

      {/* Free-Use Key Section */}
      <Card shadow="sm" padding={isMobile ? "md" : "lg"} radius="md" withBorder>
        <Stack gap="md">
          <Group gap="xs">
            <IconKey size={20} />
            <Label>{t("myOrganization.apiKeys.freeUseKey.title")}</Label>
          </Group>
          <HelperText>{t("myOrganization.apiKeys.freeUseKey.description")}</HelperText>

          {freeUse.shareId ? (
            <Group gap="sm" wrap="wrap">
              <Badge color="cyan" variant="light" size="lg">
                {freeUse.share?.apiKey?.name || t("myOrganization.apiKeys.freeUseKey.unknownKey")}
                {freeUse.share?.apiKey?.platform &&
                  ` (${state.platformName(freeUse.share.apiKey.platform)})`}
              </Badge>
              <DangerButton
                variant="outline"
                size="xs"
                onClick={freeUse.clear}
                loading={freeUse.saving}
              >
                {t("myOrganization.apiKeys.freeUseKey.remove")}
              </DangerButton>
            </Group>
          ) : (
            <Select
              placeholder={t("myOrganization.apiKeys.freeUseKey.selectPlaceholder")}
              aria-label={t("myOrganization.apiKeys.freeUseKey.title")}
              data={freeUse.options}
              onChange={freeUse.select}
              disabled={freeUse.saving}
              clearable={false}
              size="sm"
              style={{ maxWidth: 400 }}
            />
          )}
        </Stack>
      </Card>

      {/* Share Key Modal */}
      <Modal
        opened={shareModal.opened}
        onClose={shareModal.close}
        title={t("myOrganization.apiKeys.shareKeyTitle")}
      >
        <Stack gap="md">
          <Select
            label={t("myOrganization.apiKeys.selectKey")}
            placeholder={t("myOrganization.apiKeys.selectKeyPlaceholder")}
            data={shareModal.keys.map((key) => ({
              value: key.id || "",
              label: `${key.name || t("unnamed")} (${key.platform})`,
            }))}
            value={shareModal.selectedKeyId}
            onChange={shareModal.select}
          />

          <Group justify="flex-end" mt="md">
            <TextButton onClick={shareModal.close}>{t("cancel")}</TextButton>
            <ActionButton
              onClick={shareModal.submit}
              loading={shareModal.loading}
              disabled={!shareModal.selectedKeyId}
            >
              {t("myOrganization.apiKeys.share")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      {/* Auto-share personal key confirmation for free-use key */}
      <AutoShareConfirmModal
        opened={autoShareModal.opened}
        onClose={autoShareModal.close}
        onConfirm={autoShareModal.confirm}
        keyName={autoShareModal.keyName}
        orgName={institutionName ?? institutionId}
        isLoading={autoShareModal.loading}
        error={autoShareModal.error}
      />
    </Stack>
  );
}

/** The key list's columns, by key. */
function useOrgKeyColumns(state: OrgApiKeysTabState): Record<OrgKeyColumnKey, OrgKeyColumn> {
  const { t } = useTranslation("common");
  return {
    name: {
      key: "name",
      header: t("myOrganization.apiKeys.keyName"),
      sortBy: "name",
      render: (share) => (
        <Group gap="xs" wrap="nowrap">
          <IconKey size={16} />
          <BodyText size="sm">{share.apiKey?.name || t("unnamed")}</BodyText>
        </Group>
      ),
    },
    owner: {
      key: "owner",
      header: t("myOrganization.apiKeys.owner"),
      sortBy: "owner",
      render: (share) => <BodyText size="sm">{share.apiKey?.userName}</BodyText>,
    },
    platform: {
      key: "platform",
      header: t("myOrganization.apiKeys.platform"),
      sortBy: "platform",
      visibleFrom: "sm",
      render: (share) => (
        <BodyText size="sm">{state.platformName(share.apiKey?.platform)}</BodyText>
      ),
    },
    actions: {
      key: "actions",
      header: t("myOrganization.actions"),
      width: 90,
      interactive: true,
      render: (share) => <RemoveKeyButton share={share} state={state} />,
    },
  };
}

/** Removes a key from the organisation, where the caller may. */
function RemoveKeyButton({
  share,
  state,
  size,
}: {
  share: KeyShareView;
  state: OrgApiKeysTabState;
  size?: "sm";
}) {
  const { t } = useTranslation("common");
  if (!share.mayRevoke) return null;
  return (
    <DeleteIconButton
      size={size}
      onClick={() => state.remove(share)}
      loading={state.removing}
      aria-label={t("myOrganization.apiKeys.removeShare")}
    />
  );
}
