import { Badge, Box, Card, Group, Loader, Stack } from "@mantine/core";
import { modals } from "@mantine/modals";
import { IconRefresh, IconSearch } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, GenericButtonWithText } from "@components/buttons";
import { BodyText, CardTitle, HelperText, Label } from "@components/typography";
import { useSiteUpdates } from "../hooks/useSiteUpdates";

const INSTALL_COLOR: Record<string, string> = {
  installing: "blue",
  installed: "green",
  "rolled-back": "orange",
  failed: "red",
  "rollback-failed": "red",
};

/** The site's version and its upgrades, for the admin. */
export function SiteUpdates() {
  const { t } = useTranslation("common");
  const updates = useSiteUpdates();

  const confirmUpgrade = (version: string) =>
    modals.openConfirmModal({
      title: t("serverSettings.updates.upgradeTo", { version }),
      children: <BodyText size="sm">{t("serverSettings.updates.upgradeConfirm", { version })}</BodyText>,
      labels: { confirm: t("serverSettings.updates.upgrade"), cancel: t("cancel") },
      onConfirm: () => updates.requestUpgrade(version),
    });

  return (
    <Card shadow="sm" padding="lg" radius="md" withBorder>
      <Stack gap="md">
        <Group gap="xs">
          <IconRefresh size={20} />
          <CardTitle>{t("serverSettings.updates.title")}</CardTitle>
        </Group>

        {updates.loading ? (
          <Loader size="sm" />
        ) : (
          <>
            <Group gap="xs">
              <Label>{t("serverSettings.updates.running")}</Label>
              <BodyText size="sm">{updates.running || t("serverSettings.updates.noVersion")}</BodyText>
              {updates.busy && <Loader size="xs" />}
            </Group>

            {!updates.updaterEnabled && (
              <HelperText>{t("serverSettings.updates.notEnabled")}</HelperText>
            )}

            {updates.install && (
              <Group gap="xs">
                <Badge color={INSTALL_COLOR[updates.install.state]} variant="light">
                  {t(`serverSettings.updates.installState.${updates.install.state}`)}
                </Badge>
                <HelperText>
                  {t("serverSettings.updates.lastInstall", {
                    version: updates.install.version,
                    from: updates.install.from || "–",
                  })}
                </HelperText>
              </Group>
            )}

            {updates.request?.state === "rejected" && (
              <HelperText c="red">
                {t("serverSettings.updates.rejected", { version: updates.request.version })}
              </HelperText>
            )}

            {updates.updaterEnabled && updates.available.length === 0 && (
              <HelperText>{t("serverSettings.updates.upToDate")}</HelperText>
            )}

            {updates.available.map((release) => (
              <Card key={release.version} padding="sm" radius="sm" withBorder>
                <Stack gap="xs">
                  <Group justify="space-between">
                    <Label>{release.version}</Label>
                    <ActionButton
                      size="xs"
                      onClick={() => confirmUpgrade(release.version)}
                      disabled={updates.busy}
                    >
                      {t("serverSettings.updates.upgrade")}
                    </ActionButton>
                  </Group>
                  {release.notes && (
                    <Box style={{ whiteSpace: "pre-wrap" }}>
                      <BodyText size="sm">{release.notes}</BodyText>
                    </Box>
                  )}
                </Stack>
              </Card>
            ))}

            {updates.updaterEnabled && (
              <Group gap="sm" justify="space-between">
                <HelperText>
                  {updates.checking
                    ? t("serverSettings.updates.checking")
                    : updates.checked
                      ? t("serverSettings.updates.checked", {
                          time: new Date(updates.checked).toLocaleString(),
                        })
                      : t("serverSettings.updates.neverChecked")}
                </HelperText>
                <GenericButtonWithText
                  icon={updates.checking ? <Loader size={14} /> : <IconSearch size={14} />}
                  onClick={updates.checkNow}
                  disabled={updates.checking || updates.busy || updates.checkInSeconds > 0}
                  size="xs"
                  variant="light"
                >
                  {t("serverSettings.updates.checkNow")}
                </GenericButtonWithText>
              </Group>
            )}

            {updates.checkInSeconds > 0 && (
              <HelperText>
                {updates.checkRefused === "check_running"
                  ? t("serverSettings.updates.checkRunningElsewhere")
                  : t("serverSettings.updates.nextCheckIn", { seconds: updates.checkInSeconds })}
              </HelperText>
            )}

            {!updates.checking &&
              (updates.lastCheck?.state === "failed" || updates.lastCheck?.state === "rejected") && (
                <HelperText c="red">
                  {t(`serverSettings.updates.checkState.${updates.lastCheck.state}`)}
                </HelperText>
              )}
          </>
        )}
      </Stack>
    </Card>
  );
}
