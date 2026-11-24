import {
  ActionIcon,
  Badge,
  Card,
  Center,
  Divider,
  Group,
  Indicator,
  Modal,
  Popover,
  ScrollArea,
  Stack,
  UnstyledButton,
} from "@mantine/core";
import {
  IconBell,
  IconBuildingCommunity,
  IconCircleCheck,
  IconCircleX,
  IconHourglass,
  type Icon,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, CancelButton, DangerButton, TextButton } from "@components/buttons";
import { BodyText, HelperText } from "@components/typography";
import type { InviteCardView, NotificationType, NotificationView } from "@/api/types";
import { useTranslateRole } from "@/common/lib/roles";
import { formatRelativeTime } from "@/common/lib/formatters";
import { useNotificationBell, type BellInvite } from "@/common/hooks/useNotificationBell";
import classes from "./NotificationBell.module.css";

const TYPE_ICONS: Record<NotificationType, { icon: Icon; color: string }> = {
  newOrganisations: { icon: IconBuildingCommunity, color: "var(--mantine-color-accent-7)" },
  pagesWaiting: { icon: IconHourglass, color: "var(--mantine-color-accent-7)" },
  pageApproved: { icon: IconCircleCheck, color: "var(--mantine-color-green-7)" },
  pageTurnedDown: { icon: IconCircleX, color: "var(--mantine-color-red-7)" },
};

/**
 * The header's bell: grey with nothing new, red with one entry, red with
 * their number with more. It wiggles once each time there is more; its
 * panel lists the caller's notifications and invites.
 */
export function NotificationBell() {
  const { t } = useTranslation("common");
  const bell = useNotificationBell();
  const empty = bell.count === 0;

  return (
    <>
      <Popover
        opened={bell.opened}
        onClose={bell.close}
        position="bottom-end"
        withArrow
        shadow="md"
        trapFocus
      >
        <Popover.Target>
          <Indicator
            inline
            label={bell.count}
            size={18}
            offset={4}
            position="top-end"
            // white on the red bell; the theme's autoContrast writes the count dark
            color="white"
            disabled={bell.count < 2}
          >
            <ActionIcon
              variant={empty ? "subtle" : "filled"}
              color={empty ? "gray" : "red"}
              size="lg"
              radius="xl"
              onClick={bell.toggle}
              aria-label={empty ? t("notifications.title") : t("notifications.titleWithCount", { count: bell.count })}
            >
              <span key={bell.wiggles} className={bell.wiggles > 0 ? classes.wiggle : undefined}>
                <IconBell size={22} />
              </span>
            </ActionIcon>
          </Indicator>
        </Popover.Target>

        <Popover.Dropdown p="xs" className={classes.panel}>
          {empty ? (
            <Center py="md">
              <HelperText ta="center">{t("notifications.nothingNew")}</HelperText>
            </Center>
          ) : (
            <ScrollArea.Autosize mah="70vh" type="auto">
              <Stack gap="xs">
                {bell.notifications.map((n) => (
                  <NotificationEntry key={`${n.type}:${n.subject ?? ""}`} notification={n} onOpen={bell.openNotification} />
                ))}
                {bell.invites.map((entry) => (
                  <InviteEntry
                    key={entry.invite.id}
                    entry={entry}
                    accepting={bell.accepting}
                    onAccept={bell.acceptInvite}
                    onDecline={bell.askDecline}
                  />
                ))}
                {bell.notifications.length > 0 && (
                  <>
                    <Divider />
                    <Group justify="center">
                      <TextButton size="xs" onClick={bell.acknowledgeAll} disabled={bell.acknowledgingAll}>
                        {t("notifications.acknowledgeAll")}
                      </TextButton>
                    </Group>
                  </>
                )}
              </Stack>
            </ScrollArea.Autosize>
          )}
        </Popover.Dropdown>
      </Popover>

      <Modal opened={!!bell.declining} onClose={bell.cancelDecline} title={t("notifications.declineTitle")} size="sm">
        <Stack gap="md">
          <BodyText>{t("notifications.declineConfirm")}</BodyText>
          <Group justify="flex-end" gap="xs">
            <CancelButton onClick={bell.cancelDecline}>{t("cancel")}</CancelButton>
            <DangerButton onClick={bell.confirmDecline} loading={bell.decliningPending}>
              {t("notifications.decline")}
            </DangerButton>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}

/** One notification: its text, how many it folds and when; clicking leads to its target. */
function NotificationEntry({
  notification: n,
  onOpen,
}: {
  notification: NotificationView;
  onOpen: (n: NotificationView) => void;
}) {
  const { t, i18n } = useTranslation("common");
  const { icon: TypeIcon, color } = TYPE_ICONS[n.type];
  return (
    <UnstyledButton className={classes.entry} onClick={() => onOpen(n)}>
      <TypeIcon size={20} color={color} style={{ flexShrink: 0, marginTop: 2 }} />
      <Stack gap={0} style={{ flex: 1, minWidth: 0 }}>
        <BodyText size="sm">{t(`notifications.types.${n.type}`, { count: n.count })}</BodyText>
        <HelperText size="xs">{formatRelativeTime(n.at, i18n.language)}</HelperText>
      </Stack>
    </UnstyledButton>
  );
}

/** One pending invite, accepted or declined here. */
function InviteEntry({
  entry: { invite, workshop, target },
  accepting,
  onAccept,
  onDecline,
}: {
  entry: BellInvite;
  accepting: boolean;
  onAccept: (invite: InviteCardView) => void;
  onDecline: (invite: InviteCardView) => void;
}) {
  const { t } = useTranslation("common");
  const translateRole = useTranslateRole();
  return (
    <Card withBorder padding="sm">
      <Stack gap="xs">
        <Group justify="space-between" align="flex-start" wrap="nowrap">
          <Stack gap={2}>
            <BodyText size="sm" fw={500}>
              {workshop ? t("notifications.workshopInvite") : t("notifications.inviteToJoin")}
            </BodyText>
            <BodyText size="sm" fw={600}>
              {target}
            </BodyText>
          </Stack>
          {!workshop && <Badge variant="light">{translateRole(invite.role)}</Badge>}
        </Group>
        <HelperText size="sm">
          {workshop
            ? t("notifications.workshopInviteDescription")
            : t("notifications.inviteDescription", { role: translateRole(invite.role) })}
        </HelperText>
        <Group justify="flex-end" gap="xs">
          <TextButton size="xs" onClick={() => onDecline(invite)}>
            {t("notifications.decline")}
          </TextButton>
          <ActionButton size="xs" onClick={() => onAccept(invite)} loading={accepting}>
            {t("notifications.accept")}
          </ActionButton>
        </Group>
      </Stack>
    </Card>
  );
}
