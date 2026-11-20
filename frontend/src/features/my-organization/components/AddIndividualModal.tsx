import {
  Modal,
  Stack,
  Text,
  Alert,
  Group,
  ActionIcon,
  Tooltip,
  Loader,
  Center,
} from "@mantine/core";
import {
  IconUserPlus,
  IconUser,
  IconInfoCircle,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { notifications } from "@mantine/notifications";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { queryKeys } from "@/api/queryKeys";
import { useAddMemberToWorkshop } from "@/api/hooks";
import { ObjRole, type ObjUser } from "@/api/generated";

interface AddIndividualModalProps {
  opened: boolean;
  onClose: () => void;
  workshopId: string;
  institutionId: string;
}

export function AddIndividualModal({
  opened,
  onClose,
  workshopId,
  institutionId,
}: AddIndividualModalProps) {
  const { t } = useTranslation("common");
  const api = useRequiredAuthenticatedApi();
  const addMember = useAddMemberToWorkshop();

  const { data: members = [], isLoading } = useQuery<ObjUser[]>({
    queryKey: queryKeys.institutionMembers(institutionId),
    queryFn: async () => {
      const response = await api.institutions.membersList(institutionId);
      return response.data;
    },
    enabled: opened && !!institutionId,
  });

  const individuals = members.filter(
    (m) => m.role?.role === ObjRole.RoleIndividual,
  );

  const handleAdd = async (user: ObjUser) => {
    if (!user.id) return;
    try {
      await addMember.mutateAsync({ workshopId, userId: user.id });
      notifications.show({
        title: t("myOrganization.workshops.addIndividualSuccess"),
        message: t("myOrganization.workshops.addIndividualSuccessMessage", {
          name: user.name,
        }),
        color: "green",
      });
    } catch {
      notifications.show({
        title: t("error"),
        message: t("myOrganization.workshops.addIndividualError"),
        color: "red",
      });
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={t("myOrganization.workshops.addIndividualTitle")}
      size="md"
    >
      <Stack gap="md">
        <Alert icon={<IconInfoCircle size={16} />} color="blue">
          {t("myOrganization.workshops.addIndividualHint")}
        </Alert>

        <Text size="sm" c="dimmed" fs="italic">
          {t("myOrganization.workshops.addIndividualStaffNote")}
        </Text>

        {isLoading ? (
          <Center py="md">
            <Loader size="sm" />
          </Center>
        ) : individuals.length === 0 ? (
          <Text size="sm" c="dimmed" ta="center" py="md">
            {t("myOrganization.workshops.noIndividuals")}
          </Text>
        ) : (
          <Stack gap="xs">
            {individuals.map((user) => (
              <Group
                key={user.id}
                justify="space-between"
                wrap="nowrap"
                py={4}
                px="xs"
                style={{
                  borderBottom:
                    "1px solid var(--mantine-color-gray-2)",
                }}
              >
                <Group gap="xs" style={{ flex: 1, minWidth: 0 }}>
                  <IconUser size={14} color="gray" />
                  <Stack gap={0} style={{ minWidth: 0 }}>
                    <Text size="sm" fw={500} truncate>
                      {user.name || t("myOrganization.noName")}
                    </Text>
                    {user.email && (
                      <Text size="xs" c="dimmed" truncate>
                        {user.email}
                      </Text>
                    )}
                  </Stack>
                </Group>
                <Tooltip
                  label={t("myOrganization.workshops.addToWorkshop")}
                >
                  <ActionIcon
                    variant="subtle"
                    color="blue"
                    size="sm"
                    onClick={() => handleAdd(user)}
                    loading={addMember.isPending}
                  >
                    <IconUserPlus size={14} />
                  </ActionIcon>
                </Tooltip>
              </Group>
            ))}
          </Stack>
        )}
      </Stack>
    </Modal>
  );
}
