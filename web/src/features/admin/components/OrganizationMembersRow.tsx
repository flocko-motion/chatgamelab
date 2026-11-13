import { useState } from "react";
import {
  Table,
  Text,
  Badge,
  Group,
  ActionIcon,
  Collapse,
  Stack,
  Loader,
  Box,
  Tooltip,
} from "@mantine/core";
import {
  IconChevronDown,
  IconChevronRight,
  IconTrash,
  IconSend,
  IconEdit,
  IconLock,
  IconLockOpen,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/api/queryKeys";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import type { ObjInstitution, ObjUser } from "@/api/generated";
import { headNames } from "../lib/organizationHeads";
import { useTranslateRole } from "@/common/lib/roles";

interface OrganizationMembersRowProps {
  org: ObjInstitution;
  onEdit: (org: ObjInstitution) => void;
  onDelete: (org: ObjInstitution) => void;
  onInvite: (org: ObjInstitution) => void;
  onToggleLock: (org: ObjInstitution) => void;
}

/** Consent and lock state of an organisation, shared by table row and mobile card. */
export function OrganizationStatusBadges({ org }: { org: ObjInstitution }) {
  const { t, i18n } = useTranslation("common");
  const decl = org.declaration;
  const date = (iso?: string) => (iso ? new Date(iso).toLocaleDateString(i18n.language) : "");
  return (
    <Group gap={4} wrap="wrap">
      {org.disabledAt ? (
        <Tooltip label={org.disabledReason || t("admin.organizations.lockedNoReason")} withArrow>
          <Badge color="red" variant="filled" size="sm">
            {t("admin.organizations.statusLocked")}
          </Badge>
        </Tooltip>
      ) : (
        <Badge color="green" variant="light" size="sm">
          {t("admin.organizations.statusActive")}
        </Badge>
      )}
      {decl &&
        (decl.pending ? (
          <Badge color="orange" variant="light" size="sm">
            {decl.consentedVersion
              ? t("admin.organizations.consentOutdated", { version: decl.consentedVersion })
              : t("admin.organizations.consentMissing")}
          </Badge>
        ) : (
          <Badge color="gray" variant="light" size="sm">
            {t("admin.organizations.consentGiven", {
              version: decl.consentedVersion,
              date: date(decl.consentedAt),
            })}
          </Badge>
        ))}
    </Group>
  );
}


export function OrganizationMembersRow({
  org,
  onEdit,
  onDelete,
  onInvite,
  onToggleLock,
}: OrganizationMembersRowProps) {
  const { t } = useTranslation("common");
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const translateRole = useTranslateRole();
  const [expanded, setExpanded] = useState(false);

  const { data: members, isLoading: membersLoading } = useQuery({
    queryKey: queryKeys.institutionMembers(org.id!),
    queryFn: async () => {
      const response = await api.institutions.membersList(org.id!);
      return response.data;
    },
    enabled: expanded,
  });

  const removeMemberMutation = useMutation({
    mutationFn: async (userId: string) => {
      await api.institutions.membersDelete(org.id!, userId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionMembers(org.id!),
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.institutions });
    },
  });

  const handleRemoveMember = (userId: string, userName: string) => {
    if (
      confirm(t("admin.organizations.removeMemberConfirm", { name: userName }))
    ) {
      removeMemberMutation.mutate(userId);
    }
  };

  return (
    <>
      <Table.Tr
        style={{ cursor: "pointer" }}
        onClick={() => setExpanded(!expanded)}
      >
        <Table.Td>
          <Group gap="xs">
            <ActionIcon variant="subtle" size="sm">
              {expanded ? (
                <IconChevronDown size={16} />
              ) : (
                <IconChevronRight size={16} />
              )}
            </ActionIcon>
            <Stack gap={0}>
              <Text fw={500}>{org.name}</Text>
              {(org.operator || org.city) && (
                <Text size="xs" c="dimmed">
                  {[org.operator, org.city].filter(Boolean).join(" · ")}
                </Text>
              )}
            </Stack>
          </Group>
        </Table.Td>
        <Table.Td>
          <Text size="sm">{headNames(org) || "–"}</Text>
        </Table.Td>
        <Table.Td>
          <OrganizationStatusBadges org={org} />
        </Table.Td>
        <Table.Td>
          <Badge variant="light" color="blue">
            {org.members?.length || 0} {t("admin.organizations.membersCount")}
          </Badge>
        </Table.Td>
        <Table.Td onClick={(e) => e.stopPropagation()}>
          <Group gap="xs">
            <ActionIcon
              variant="subtle"
              color="blue"
              onClick={() => onInvite(org)}
              title={t("admin.organizations.sendInvite")}
            >
              <IconSend size={16} />
            </ActionIcon>
            <ActionIcon
              variant="subtle"
              color="gray"
              onClick={() => onEdit(org)}
              title={t("edit")}
            >
              <IconEdit size={16} />
            </ActionIcon>
            <ActionIcon
              variant="subtle"
              color={org.disabledAt ? "green" : "orange"}
              onClick={() => onToggleLock(org)}
              title={org.disabledAt ? t("admin.organizations.unlock") : t("admin.organizations.lock")}
            >
              {org.disabledAt ? <IconLockOpen size={16} /> : <IconLock size={16} />}
            </ActionIcon>
            <ActionIcon
              variant="subtle"
              color="red"
              onClick={() => onDelete(org)}
              title={t("delete")}
            >
              <IconTrash size={16} />
            </ActionIcon>
          </Group>
        </Table.Td>
      </Table.Tr>
      {expanded && (
        <Table.Tr>
          <Table.Td colSpan={5} style={{ padding: 0 }}>
            <Collapse in={expanded}>
              <Box bg="gray.0" p="md">
                {membersLoading ? (
                  <Group justify="center" p="md">
                    <Loader size="sm" />
                  </Group>
                ) : members && members.length > 0 ? (
                  <Stack gap="xs">
                    <Text size="sm" fw={600} c="dimmed">
                      {t("admin.organizations.membersList")}
                    </Text>
                    <Table>
                      <Table.Thead>
                        <Table.Tr>
                          <Table.Th>
                            {t("admin.organizations.memberName")}
                          </Table.Th>
                          <Table.Th>
                            {t("admin.organizations.memberEmail")}
                          </Table.Th>
                          <Table.Th>
                            {t("admin.organizations.memberRole")}
                          </Table.Th>
                          <Table.Th style={{ width: 80 }}>
                            {t("admin.organizations.actions")}
                          </Table.Th>
                        </Table.Tr>
                      </Table.Thead>
                      <Table.Tbody>
                        {members.map((member: ObjUser) => (
                          <Table.Tr key={member.id}>
                            <Table.Td>
                              <Text size="sm">{member.name || "-"}</Text>
                            </Table.Td>
                            <Table.Td>
                              <Text size="sm" c="dimmed">
                                {member.email || "-"}
                              </Text>
                            </Table.Td>
                            <Table.Td>
                              <Badge size="sm" variant="light">
                                {translateRole(member.role?.role)}
                              </Badge>
                            </Table.Td>
                            <Table.Td>
                              <ActionIcon
                                variant="subtle"
                                color="red"
                                size="sm"
                                onClick={() =>
                                  handleRemoveMember(
                                    member.id!,
                                    member.name || member.email || "this user",
                                  )
                                }
                                loading={removeMemberMutation.isPending}
                                title={t("admin.organizations.removeMember")}
                              >
                                <IconTrash size={14} />
                              </ActionIcon>
                            </Table.Td>
                          </Table.Tr>
                        ))}
                      </Table.Tbody>
                    </Table>
                  </Stack>
                ) : (
                  <Text size="sm" c="dimmed" ta="center" py="md">
                    {t("admin.organizations.noMembers")}
                  </Text>
                )}
              </Box>
            </Collapse>
          </Table.Td>
        </Table.Tr>
      )}
    </>
  );
}
