import { useEffect, useMemo, useRef, useState } from "react";
import {
  ActionIcon,
  Button,
  Card,
  Group,
  Indicator,
  Modal,
  Stack,
  Text,
  Tooltip,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconBuildingCommunity } from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { ROUTES } from "@/common/routes/routes";
import {
  INSTITUTION_APPLICATIONS_QUERY_KEY,
  getSeenApplicationIds,
  isNewOrganisation,
  markApplicationsSeen,
} from "../lib/institutionApplications";

/** How often the admin header checks for new organisation registrations. */
const REFRESH_MS = 2 * 60 * 1000;

/**
 * Header icon for admins: shows organisations registered in the last two weeks that this admin
 * has not looked at yet, and opens a notice once per new one (like the invite notice for users).
 * Stage 1 has no e-mail confirmation, so admins check new organisations and can lock them.
 */
export function AdminRegistrationBell() {
  const { t, i18n } = useTranslation("common");
  const api = useRequiredAuthenticatedApi();
  const navigate = useNavigate();
  const [opened, { open, close }] = useDisclosure(false);
  const hasAutoOpened = useRef(false);

  const { data, isLoading } = useQuery({
    queryKey: INSTITUTION_APPLICATIONS_QUERY_KEY,
    queryFn: async () =>
      (await api.institutionApplications.institutionApplicationsList()).data,
    refetchInterval: REFRESH_MS,
  });

  // Bumped when the admin closes the notice, so the badge updates right away.
  const [seenVersion, setSeenVersion] = useState(0);
  const seen = useMemo(() => getSeenApplicationIds(), [seenVersion]); // eslint-disable-line react-hooks/exhaustive-deps
  const newOrgs = useMemo(
    () => (data ?? []).filter((a) => isNewOrganisation(a)),
    [data],
  );
  // The badge counts organisations this admin has not been told about yet;
  // the notice lists all organisations of the last two weeks.
  const unseenCount = newOrgs.filter((a) => a.id && !seen.has(a.id)).length;
  const hasUnseen = unseenCount > 0;

  useEffect(() => {
    if (!isLoading && hasUnseen && !hasAutoOpened.current && !opened) {
      hasAutoOpened.current = true;
      open();
    }
  }, [isLoading, hasUnseen, opened, open]);

  const handleClose = () => {
    markApplicationsSeen(newOrgs.map((a) => a.id).filter(Boolean) as string[]);
    setSeenVersion((v) => v + 1);
    close();
  };

  const goToOverview = () => {
    handleClose();
    navigate({ to: ROUTES.ADMIN_ORGANIZATIONS as "/" });
  };

  const date = (iso?: string) =>
    iso
      ? new Date(iso).toLocaleString(i18n.language, {
          dateStyle: "short",
          timeStyle: "short",
        })
      : "";

  const count = newOrgs.length;

  return (
    <>
      <Tooltip label={t("admin.applications.bell.label")} withArrow>
        <Indicator
          inline
          size={18}
          offset={4}
          position="top-end"
          color="red"
          label={unseenCount}
          disabled={unseenCount === 0}
          processing={unseenCount > 0}
        >
          <ActionIcon
            variant="subtle"
            color="white"
            size="lg"
            onClick={open}
            aria-label={t("admin.applications.bell.label")}
          >
            <IconBuildingCommunity size={22} />
          </ActionIcon>
        </Indicator>
      </Tooltip>

      <Modal
        opened={opened}
        onClose={handleClose}
        title={t("admin.applications.bell.title")}
        size="md"
      >
        {count === 0 ? (
          <Text c="dimmed" ta="center" py="xl">
            {t("admin.applications.bell.empty")}
          </Text>
        ) : (
          <Stack gap="md">
            <Text size="sm">
              {t("admin.applications.bell.intro", { count })}
            </Text>
            {newOrgs.map((a) => (
              <Card key={a.id} withBorder padding="sm">
                <Stack gap={2}>
                  <Text fw={600}>{a.institutionName}</Text>
                  <Text size="sm">
                    {a.personName}
                    {a.personFunction ? ` (${a.personFunction})` : ""}
                    {a.email ? ` · ${a.email}` : ""}
                  </Text>
                  <Text size="xs" c="dimmed">
                    {[a.city, date(a.createdAt)].filter(Boolean).join(" · ")}
                  </Text>
                </Stack>
              </Card>
            ))}
            <Text size="xs" c="dimmed">
              {t("admin.applications.bell.lockHint")}
            </Text>
            <Group justify="flex-end" gap="xs">
              <Button variant="subtle" onClick={handleClose}>
                {t("admin.applications.bell.later")}
              </Button>
              <Button onClick={goToOverview}>
                {t("admin.applications.bell.open")}
              </Button>
            </Group>
          </Stack>
        )}
      </Modal>
    </>
  );
}
