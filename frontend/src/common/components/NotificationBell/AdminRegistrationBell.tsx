import { ActionIcon, Card, Group, Indicator, Modal, Stack, Tooltip } from "@mantine/core";
import { IconBuildingCommunity } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { BodyText, HelperText } from "@components/typography";
import { ROUTES } from "@/common/routes/routes";
import { formatDateTime } from "@/common/lib/formatters";
import { useNewOrganisationsNotice } from "@/common/hooks/useNewOrganisationsNotice";

/**
 * The header's icon for admins: the organisations that registered themselves
 * in the last two weeks this admin has not looked at, with a notice that
 * opens once per new one. Organisations are active at once, so admins check
 * them and suspend one if something is wrong.
 */
export function AdminRegistrationBell() {
  const { t, i18n } = useTranslation("common");
  const navigate = useNavigate();
  const notice = useNewOrganisationsNotice();
  const count = notice.organisations.length;

  const goToOverview = () => {
    notice.dismiss();
    navigate({ to: ROUTES.ADMIN_ORGANIZATIONS });
  };

  return (
    <>
      <Tooltip label={t("admin.registrations.bell.label")} withArrow>
        <Indicator
          inline
          size={18}
          offset={4}
          position="top-end"
          color="red"
          label={notice.unseen}
          disabled={notice.unseen === 0}
          processing={notice.unseen > 0}
        >
          <ActionIcon
            variant="subtle"
            color="white"
            size="lg"
            onClick={notice.open}
            aria-label={t("admin.registrations.bell.label")}
          >
            <IconBuildingCommunity size={22} />
          </ActionIcon>
        </Indicator>
      </Tooltip>

      <Modal opened={notice.opened} onClose={notice.dismiss} title={t("admin.registrations.bell.title")} size="md">
        {count === 0 ? (
          <HelperText ta="center">{t("admin.registrations.bell.empty")}</HelperText>
        ) : (
          <Stack gap="md">
            <BodyText size="sm">{t("admin.registrations.bell.intro", { count })}</BodyText>
            {notice.organisations.map((org) => (
              <Card key={org.id} withBorder padding="sm">
                <Stack gap={2}>
                  <BodyText fw={600}>{org.name}</BodyText>
                  {org.registration && (
                    <BodyText size="sm">
                      {`${org.registration.personName} (${org.registration.personFunction}) · ${org.registration.email}`}
                    </BodyText>
                  )}
                  <HelperText>
                    {[org.city, formatDateTime(org.registration?.at, i18n.language)].filter(Boolean).join(" · ")}
                  </HelperText>
                </Stack>
              </Card>
            ))}
            <HelperText>{t("admin.registrations.bell.suspendHint")}</HelperText>
            <Group justify="flex-end" gap="xs">
              <TextButton onClick={notice.dismiss}>{t("admin.registrations.bell.later")}</TextButton>
              <ActionButton onClick={goToOverview}>{t("admin.registrations.bell.open")}</ActionButton>
            </Group>
          </Stack>
        )}
      </Modal>
    </>
  );
}
