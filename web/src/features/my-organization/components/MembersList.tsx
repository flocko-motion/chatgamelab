import { useMemo, useState, useCallback } from "react";
import {
  Card,
  Group,
  TextInput,
  Table,
  Text,
  Stack,
  Badge,
  ActionIcon,
  Tooltip,
  LoadingOverlay,
} from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import {
  IconSearch,
  IconCrown,
  IconCrownOff,
  IconTrash,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import {
  SortSelector,
  type SortOption,
  ExpandableSearch,
} from "@/common/components/controls";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { parseSortValue } from "@/common/lib/sort";
import type { ObjUser } from "@/api/generated";

type SortField = "name" | "email" | "role";

const getRoleColor = (role?: string): string => {
  switch (role) {
    case "head":
      return "violet";
    case "staff":
      return "blue";
    case "participant":
      return "gray";
    default:
      return "gray";
  }
};

export interface MembersListProps {
  members: ObjUser[];
  isLoading: boolean;
  currentUserId?: string;
  canSeeEmails: boolean;
  isHead: boolean;
  isStaff: boolean;
  onPromote?: (user: ObjUser) => void;
  onDemote?: (user: ObjUser) => void;
  onRemove?: (user: ObjUser) => void;
}

export function MembersList({
  members,
  isLoading,
  currentUserId,
  canSeeEmails,
  isHead,
  isStaff,
  onPromote,
  onDemote,
  onRemove,
}: MembersListProps) {
  const { t } = useTranslation("common");
  const { t: tAuth } = useTranslation("auth");
  const { isMobile } = useResponsiveDesign();

  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);
  const [sortValue, setSortValue] = useState("name-asc");

  const translateRole = useCallback(
    (role?: string) => {
      if (!role) return t("myOrganization.noRole");
      const roleKey = role.toLowerCase();
      return tAuth(`profile.roles.${roleKey}`, role);
    },
    [t, tAuth],
  );

  const isCurrentUser = useCallback(
    (userId?: string) => {
      return userId === currentUserId;
    },
    [currentUserId],
  );

  const canRemoveMember = useCallback(
    (member: ObjUser) => {
      if (isCurrentUser(member.id)) return false;
      const memberRole = member.role?.role;
      if (isHead) {
        return (
          memberRole === "staff" || memberRole === "participant" || !memberRole
        );
      }
      if (isStaff) {
        return memberRole === "participant" || !memberRole;
      }
      return false;
    },
    [isHead, isStaff, isCurrentUser],
  );

  const [sortField, sortDirection] = parseSortValue<SortField>(sortValue);

  const sortOptions: SortOption[] = useMemo(
    () => [
      { value: "name-asc", label: t("myOrganization.sort.name-asc") },
      { value: "name-desc", label: t("myOrganization.sort.name-desc") },
      { value: "role-asc", label: t("myOrganization.sort.role-asc") },
      { value: "role-desc", label: t("myOrganization.sort.role-desc") },
    ],
    [t],
  );

  const filteredMembers = useMemo(() => {
    if (!members) return [];

    return members.filter((member) => {
      if (debouncedSearch) {
        const search = debouncedSearch.toLowerCase();
        const name = (member.name || "").toLowerCase();
        const email = (member.email || "").toLowerCase();
        const role = (member.role?.role || "").toLowerCase();

        return (
          name.includes(search) ||
          email.includes(search) ||
          role.includes(search)
        );
      }
      return true;
    });
  }, [members, debouncedSearch]);

  const sortedMembers = useMemo(() => {
    if (filteredMembers.length === 0) return [];

    return [...filteredMembers].sort((a, b) => {
      let aVal: string;
      let bVal: string;

      switch (sortField) {
        case "name":
          aVal = a.name || "";
          bVal = b.name || "";
          break;
        case "email":
          aVal = a.email || "";
          bVal = b.email || "";
          break;
        case "role":
          aVal = a.role?.role || "";
          bVal = b.role?.role || "";
          break;
        default:
          return 0;
      }

      const comparison = aVal.localeCompare(bVal);
      return sortDirection === "asc" ? comparison : -comparison;
    });
  }, [filteredMembers, sortField, sortDirection]);

  const showActions = isHead || isStaff;

  return (
    <Card withBorder pos="relative" p={isMobile ? "sm" : undefined}>
      <LoadingOverlay visible={isLoading} />

      <Stack gap="md">
        {isMobile ? (
          <Group gap="sm" wrap="nowrap">
            <ExpandableSearch
              value={searchQuery}
              onChange={setSearchQuery}
              placeholder={t("myOrganization.searchPlaceholder")}
            />
            <SortSelector
              options={sortOptions}
              value={sortValue}
              onChange={setSortValue}
              label={t("myOrganization.sort.label")}
            />
          </Group>
        ) : (
          <Group justify="space-between" gap="md">
            <TextInput
              placeholder={t("myOrganization.searchPlaceholder")}
              leftSection={<IconSearch size={16} />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.currentTarget.value)}
              style={{ flex: 1, maxWidth: 400 }}
            />
            <SortSelector
              options={sortOptions}
              value={sortValue}
              onChange={setSortValue}
              label={t("myOrganization.sort.label")}
              width={200}
            />
          </Group>
        )}

        <Text size="sm" c="dimmed">
          {t("myOrganization.showing", { count: sortedMembers.length })}
        </Text>

        {isMobile ? (
          <Stack gap="sm">
            {sortedMembers.map((member) => (
              <Card
                key={member.id}
                withBorder
                padding="sm"
                radius="md"
                bg={isCurrentUser(member.id) ? "violet.0" : undefined}
              >
                <Stack gap="xs">
                  <Group justify="space-between" wrap="nowrap">
                    <Group gap="xs">
                      <Text fw={600} lineClamp={1}>
                        {member.name}
                      </Text>
                      {isCurrentUser(member.id) && (
                        <Badge size="xs" variant="filled" color="violet">
                          {t("myOrganization.me")}
                        </Badge>
                      )}
                    </Group>
                    {member.role?.role && (
                      <Badge
                        color={getRoleColor(member.role.role)}
                        variant="light"
                        size="sm"
                      >
                        {translateRole(member.role.role)}
                      </Badge>
                    )}
                  </Group>
                  {canSeeEmails && member.email && (
                    <Text size="xs" c="dimmed">
                      {member.email}
                    </Text>
                  )}
                  {!isCurrentUser(member.id) && (
                    <Group gap="xs" justify="flex-end">
                      {isHead && member.role?.role === "staff" && onPromote && (
                        <Tooltip label={t("myOrganization.promoteToHead")}>
                          <ActionIcon
                            variant="light"
                            color="violet"
                            size="sm"
                            onClick={() => onPromote(member)}
                          >
                            <IconCrown size={14} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                      {isHead && member.role?.role === "head" && onDemote && (
                        <Tooltip label={t("myOrganization.demoteToStaff")}>
                          <ActionIcon
                            variant="light"
                            color="gray"
                            size="sm"
                            onClick={() => onDemote(member)}
                          >
                            <IconCrownOff size={14} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                      {canRemoveMember(member) && onRemove && (
                        <Tooltip label={t("myOrganization.removeMember")}>
                          <ActionIcon
                            variant="light"
                            color="red"
                            size="sm"
                            onClick={() => onRemove(member)}
                          >
                            <IconTrash size={14} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                    </Group>
                  )}
                </Stack>
              </Card>
            ))}
          </Stack>
        ) : (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>{t("myOrganization.name")}</Table.Th>
                {canSeeEmails && (
                  <Table.Th>{t("myOrganization.email")}</Table.Th>
                )}
                <Table.Th>{t("myOrganization.role")}</Table.Th>
                {showActions && (
                  <Table.Th style={{ width: 100 }}>
                    {t("myOrganization.actions")}
                  </Table.Th>
                )}
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {sortedMembers.map((member) => (
                <Table.Tr
                  key={member.id}
                  bg={isCurrentUser(member.id) ? "violet.0" : undefined}
                >
                  <Table.Td>
                    <Group gap="xs">
                      <Text size="sm">{member.name}</Text>
                      {isCurrentUser(member.id) && (
                        <Badge size="xs" variant="filled" color="violet">
                          {t("myOrganization.me")}
                        </Badge>
                      )}
                    </Group>
                  </Table.Td>
                  {canSeeEmails && (
                    <Table.Td>
                      <Text size="sm" c="dimmed">
                        {member.email || "-"}
                      </Text>
                    </Table.Td>
                  )}
                  <Table.Td>
                    <Badge
                      color={getRoleColor(member.role?.role)}
                      variant="light"
                      size="sm"
                    >
                      {translateRole(member.role?.role)}
                    </Badge>
                  </Table.Td>
                  {showActions && (
                    <Table.Td>
                      <Group gap="xs">
                        {!isCurrentUser(member.id) &&
                          isHead &&
                          member.role?.role === "staff" &&
                          onPromote && (
                            <Tooltip label={t("myOrganization.promoteToHead")}>
                              <ActionIcon
                                variant="subtle"
                                color="violet"
                                onClick={() => onPromote(member)}
                              >
                                <IconCrown size={16} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                        {!isCurrentUser(member.id) &&
                          isHead &&
                          member.role?.role === "head" &&
                          onDemote && (
                            <Tooltip label={t("myOrganization.demoteToStaff")}>
                              <ActionIcon
                                variant="subtle"
                                color="gray"
                                onClick={() => onDemote(member)}
                              >
                                <IconCrownOff size={16} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                        {!isCurrentUser(member.id) &&
                          canRemoveMember(member) &&
                          onRemove && (
                            <Tooltip label={t("myOrganization.removeMember")}>
                              <ActionIcon
                                variant="subtle"
                                color="red"
                                onClick={() => onRemove(member)}
                              >
                                <IconTrash size={16} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                      </Group>
                    </Table.Td>
                  )}
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}

        {sortedMembers.length === 0 && !isLoading && (
          <Text c="dimmed" ta="center" py="xl">
            {debouncedSearch
              ? t("myOrganization.noResults")
              : t("myOrganization.empty")}
          </Text>
        )}
      </Stack>
    </Card>
  );
}
