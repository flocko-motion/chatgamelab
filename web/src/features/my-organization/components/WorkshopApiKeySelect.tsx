import { useState, useMemo } from "react";
import { Select, Modal, Stack, Text, Group, Button } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconKey, IconInfoCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useApiKeys, useInstitutionApiKeys } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";

/** Value prefix to distinguish personal keys from org shares in the select */
const PERSONAL_KEY_PREFIX = "personal:";

interface WorkshopApiKeySelectProps {
  institutionId: string;
  workshopId: string;
  value: string | null;
  onChange: (data: {
    apiKeyShareId?: string | null;
    apiKeyId?: string | null;
  }) => void;
  disabled?: boolean;
  size?: "xs" | "sm" | "md";
}

export function WorkshopApiKeySelect({
  institutionId,
  value,
  onChange,
  disabled = false,
  size = "sm",
}: WorkshopApiKeySelectProps) {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();
  const [pendingPersonalKey, setPendingPersonalKey] = useState<{
    apiKeyId: string;
    name: string;
  } | null>(null);
  const [confirmOpened, { open: openConfirm, close: closeConfirm }] =
    useDisclosure(false);

  const { data: institutionApiKeys } = useInstitutionApiKeys(institutionId);
  const { data: apiKeysData } = useApiKeys();
  const currentUserId = backendUser?.id;

  const { options, isCurrentValuePersonal } = useMemo(() => {
    // Separate org shares from workshop-specific shares
    const orgShares =
      institutionApiKeys?.filter((key) => key.institution && !key.workshop) || [];
    const workshopShares =
      institutionApiKeys?.filter((key) => key.workshop && !key.institution) || [];

    const orgOptions = orgShares.map((key) => ({
      value: key.id || "",
      label: `${key.apiKey?.name || key.apiKey?.platform || "Unknown"} — ${key.apiKey?.platform || ""}`,
    }));

    // Track which underlying API keys are already shared with the org
    const orgKeyIds = new Set(orgShares.map((key) => key.apiKeyId));

    // Track which API keys are already assigned to THIS workshop as workshop-specific shares
    const workshopKeyIds = new Set(workshopShares.map((key) => key.apiKeyId));

    // Personal keys: user's own keys that are NOT already shared with the org
    // AND not already assigned to this workshop as a workshop-specific share
    const personalOptions =
      apiKeysData?.apiKeys
        ?.filter((key) => !orgKeyIds.has(key.id) && !workshopKeyIds.has(key.id))
        .map((key) => ({
          value: `${PERSONAL_KEY_PREFIX}${key.id}`,
          label: `${key.name || key.platform || "Unknown"} — ${key.platform || ""}`,
        })) || [];

    const groups = [
      {
        group: t("myOrganization.workshops.apiKeyGroupOrg"),
        items: orgOptions,
      },
    ];

    if (personalOptions.length > 0) {
      groups.push({
        group: t("myOrganization.workshops.apiKeyGroupPersonal"),
        items: personalOptions,
      });
    }

    // If the current value is a workshop-specific share, add it as an option
    // so the Select can display it properly
    const isPersonal =
      !!value &&
      !orgOptions.some((o) => o.value === value) &&
      value !== "";

    if (isPersonal) {
      const workshopShare = workshopShares.find((s) => s.id === value);
      if (workshopShare) {
        const ownerName = workshopShare.apiKey?.userName || workshopShare.user?.name || "";
        const keyName = workshopShare.apiKey?.name || workshopShare.apiKey?.platform || "Unknown";
        const isOwnedByCurrentUser = workshopShare.apiKey?.userId === currentUserId;
        
        // Determine the group based on ownership
        const groupLabel = isOwnedByCurrentUser
          ? t("myOrganization.workshops.apiKeyGroupPersonal")
          : t("myOrganization.workshops.apiKeyGroupColleague");
        
        // Hide key name from non-owners - only show "Personal key from [name]"
        const displayLabel = !isOwnedByCurrentUser && ownerName
          ? t("myOrganization.workshops.colleagueKeyLabel", { name: ownerName })
          : `${keyName} — ${workshopShare.apiKey?.platform || ""}`;
        
        groups.unshift({
          group: groupLabel,
          items: [
            {
              value: workshopShare.id || "",
              label: displayLabel,
            },
          ],
        });
      }
    }

    return { options: groups, isCurrentValuePersonal: isPersonal };
  }, [institutionApiKeys, apiKeysData, value, currentUserId, t]);

  const handleChange = (selectedValue: string | null) => {
    if (!selectedValue || selectedValue === "") {
      onChange({ apiKeyShareId: null });
      return;
    }

    if (selectedValue.startsWith(PERSONAL_KEY_PREFIX)) {
      const apiKeyId = selectedValue.slice(PERSONAL_KEY_PREFIX.length);
      const key = apiKeysData?.apiKeys?.find((k) => k.id === apiKeyId);
      setPendingPersonalKey({
        apiKeyId,
        name: key?.name || key?.platform || "Unknown",
      });
      openConfirm();
      return;
    }

    onChange({ apiKeyShareId: selectedValue });
  };

  const handleConfirmPersonalKey = () => {
    if (pendingPersonalKey) {
      onChange({ apiKeyId: pendingPersonalKey.apiKeyId });
    }
    closeConfirm();
    setPendingPersonalKey(null);
  };

  const handleCancelPersonalKey = () => {
    closeConfirm();
    setPendingPersonalKey(null);
  };

  return (
    <>
      <Select
        size={size}
        data={options}
        value={value || ""}
        onChange={handleChange}
        placeholder={t("myOrganization.workshops.selectApiKey")}
        clearable
        disabled={disabled}
        leftSection={<IconKey size={14} />}
        description={
          isCurrentValuePersonal
            ? t("myOrganization.workshops.selectApiKeyPersonal")
            : undefined
        }
      />

      <Modal
        opened={confirmOpened}
        onClose={handleCancelPersonalKey}
        title={t("myOrganization.workshops.personalKeyConfirmTitle")}
        size="md"
      >
        <Stack gap="md">
          <Group gap="xs">
            <IconInfoCircle size={20} color="var(--mantine-color-blue-6)" />
            <Text fw={500}>{pendingPersonalKey?.name}</Text>
          </Group>
          <Text size="sm">
            {t("myOrganization.workshops.personalKeyConfirmMessage")}
          </Text>
          <Group justify="flex-end" mt="sm">
            <Button variant="subtle" onClick={handleCancelPersonalKey}>
              {t("cancel")}
            </Button>
            <Button onClick={handleConfirmPersonalKey}>
              {t("myOrganization.workshops.personalKeyConfirmButton")}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
