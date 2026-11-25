import { useState } from "react";
import { Alert, Container, Group, Loader, Stack, Tabs } from "@mantine/core";
import { IconAlertCircle, IconBrush, IconCpu, IconRefresh, IconShieldLock, IconSparkles } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { PageTitle } from "@components/typography";
import { useAuth } from "@/providers/AuthProvider";
import { isAdmin } from "@/common/lib/roles";
import { useServerSettings } from "../hooks/useServerSettings";
import { SiteUpdates } from "./SiteUpdates";
import { SessionRetentionSettings } from "./SessionRetentionSettings";
import { SiteBranding } from "./SiteBranding";
import { AiSettings } from "./AiSettings";
import { YouthProtectionSettings } from "./YouthProtectionSettings";
import { EngineModels } from "./EngineModels";

/** The site's settings, for the admin, one topic per tab. */
export function ServerSettings() {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();
  const state = useServerSettings();
  const [tab, setTab] = useState<string | null>("general");

  if (!isAdmin(backendUser)) {
    return (
      <Container size="xl" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title={t("error")} color="red">
          {t("serverSettings.notAuthorized")}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="lg" py="xl">
      <Stack gap="lg">
        <PageTitle>{t("serverSettings.title")}</PageTitle>

        <Tabs value={tab} onChange={setTab} keepMounted={false}>
          <Tabs.List>
            <Tabs.Tab value="general" leftSection={<IconRefresh size={16} />}>
              {t("serverSettings.tabs.general")}
            </Tabs.Tab>
            <Tabs.Tab value="branding" leftSection={<IconBrush size={16} />}>
              {t("serverSettings.tabs.branding")}
            </Tabs.Tab>
            <Tabs.Tab value="ai" leftSection={<IconSparkles size={16} />}>
              {t("serverSettings.tabs.ai")}
            </Tabs.Tab>
            <Tabs.Tab value="models" leftSection={<IconCpu size={16} />}>
              {t("serverSettings.tabs.models")}
            </Tabs.Tab>
            <Tabs.Tab value="youth" leftSection={<IconShieldLock size={16} />}>
              {t("serverSettings.tabs.youthProtection")}
            </Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="general" pt="lg">
            <Stack gap="lg">
              <SiteUpdates />
              {state.loading ? <Centered /> : <SessionRetentionSettings state={state} />}
            </Stack>
          </Tabs.Panel>
          <Tabs.Panel value="branding" pt="lg">
            <SiteBranding />
          </Tabs.Panel>
          <Tabs.Panel value="ai" pt="lg">
            {state.loading ? <Centered /> : <AiSettings state={state} />}
          </Tabs.Panel>
          <Tabs.Panel value="models" pt="lg">
            <EngineModels />
          </Tabs.Panel>
          <Tabs.Panel value="youth" pt="lg">
            {state.loading ? <Centered /> : <YouthProtectionSettings state={state} />}
          </Tabs.Panel>
        </Tabs>
      </Stack>
    </Container>
  );
}

function Centered() {
  return (
    <Group justify="center" py="xl">
      <Loader />
    </Group>
  );
}
