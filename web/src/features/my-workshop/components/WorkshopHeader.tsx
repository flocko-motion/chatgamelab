import {
  Group,
  Text,
  Badge,
  HoverCard,
  Stack,
  Divider,
  ActionIcon,
  Tooltip,
} from "@mantine/core";
import {
  IconUsers,
  IconPlayerPause,
  IconPlayerPlay,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { PageTitle } from "@components/typography";
import { useWorkshop, useUpdateWorkshop } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";

interface WorkshopHeaderProps {
  workshopName?: string;
  organizationName?: string;
  workshopId?: string;
  showMembers?: boolean;
}

export function WorkshopHeader({
  workshopName,
  organizationName,
  workshopId,
  showMembers = false,
}: WorkshopHeaderProps) {
  const { t } = useTranslation("myWorkshop");
  const { t: tCommon } = useTranslation("common");
  const { retryBackendFetch } = useAuth();
  const { isMobile } = useResponsiveDesign();
  const { data: workshop } = useWorkshop(showMembers ? workshopId : undefined);
  const updateWorkshop = useUpdateWorkshop();

  const participants = workshop?.participants ?? [];
  const memberCount = participants.length;
  const isPaused = workshop?.isPaused ?? false;

  const handleTogglePause = async () => {
    if (!workshop?.id) return;
    await updateWorkshop.mutateAsync({
      id: workshop.id,
      name: workshop.name || "",
      active: workshop.active ?? true,
      public: workshop.public ?? false,
      showPublicGames: workshop.showPublicGames ?? false,
      showOtherParticipantsGames: workshop.showOtherParticipantsGames ?? true,
      designEditingEnabled: workshop.designEditingEnabled ?? false,
      aiQualityTier: workshop.aiQualityTier ?? undefined,
      isPaused: !isPaused,
    });
    retryBackendFetch();
  };

  return (
    <>
      <Group gap="md" align="center">
        <PageTitle>{workshopName || t("title")}</PageTitle>
        {showMembers && (
          <Tooltip
            label={
              isPaused
                ? tCommon("myOrganization.workshops.unpause")
                : tCommon("myOrganization.workshops.isPaused")
            }
            disabled={!isMobile}
          >
            <ActionIcon
              variant={isPaused ? "filled" : "subtle"}
              color={isPaused ? "orange" : "gray"}
              size={isMobile ? "lg" : "xl"}
              onClick={handleTogglePause}
              loading={updateWorkshop.isPending}
              style={
                isMobile ? undefined : { width: "auto", paddingInline: 12 }
              }
            >
              {isMobile ? (
                isPaused ? (
                  <IconPlayerPlay size={18} />
                ) : (
                  <IconPlayerPause size={18} />
                )
              ) : (
                <Group gap={6} wrap="nowrap">
                  {isPaused ? (
                    <IconPlayerPlay size={18} />
                  ) : (
                    <IconPlayerPause size={18} />
                  )}
                  <Text size="sm" fw={500} c={isPaused ? "white" : undefined}>
                    {isPaused
                      ? tCommon("myOrganization.workshops.unpause")
                      : tCommon("myOrganization.workshops.isPaused")}
                  </Text>
                </Group>
              )}
            </ActionIcon>
          </Tooltip>
        )}
        {showMembers && (
          <HoverCard
            width={260}
            shadow="md"
            withArrow
            openDelay={200}
            closeDelay={100}
          >
            <HoverCard.Target>
              <Badge
                size="lg"
                variant="light"
                color="gray"
                leftSection={<IconUsers size={14} />}
                style={{ cursor: "default" }}
              >
                {memberCount}
              </Badge>
            </HoverCard.Target>
            <HoverCard.Dropdown>
              <Text size="sm" fw={600} mb={4}>
                {t("members.title")}
              </Text>
              <Divider mb="xs" />
              {memberCount === 0 ? (
                <Text size="sm" c="dimmed">
                  {t("members.empty")}
                </Text>
              ) : (
                <Stack gap={4} mah={240} style={{ overflowY: "auto" }}>
                  {participants.map((p) => (
                    <Group key={p.id} gap="xs" justify="space-between">
                      <Text
                        size="sm"
                        lineClamp={1}
                        style={{ flex: 1, minWidth: 0 }}
                      >
                        {p.name}
                      </Text>
                      <Badge
                        size="xs"
                        variant="light"
                        color={getRoleBadgeColor(p.role)}
                      >
                        {p.role}
                      </Badge>
                    </Group>
                  ))}
                </Stack>
              )}
            </HoverCard.Dropdown>
          </HoverCard>
        )}
      </Group>
      {organizationName && (
        <Text size="sm" c="dimmed">
          {t("organizator", { name: organizationName })}
        </Text>
      )}
    </>
  );
}

function getRoleBadgeColor(role?: string): string {
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
}
