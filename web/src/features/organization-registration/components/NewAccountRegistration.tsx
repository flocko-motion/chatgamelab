import { useEffect, useState } from "react";
import { Card, Container, Stack, Text } from "@mantine/core";
import { IconBuilding, IconUser } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { SectionTitle, BodyText } from "@components/typography";
import { RegistrationForm } from "@/features/auth";
import { useAuth, type RegistrationData } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { OrganizationRegistrationForm } from "./OrganizationRegistrationForm";
import {
  clearRegistrationIntent,
  getRegistrationIntent,
  type RegistrationIntent,
} from "../lib/registrationIntent";
import { notifyOrganisationRegistered } from "../lib/notifyRegistered";

/**
 * Shown after the first login of a person without an account. If they chose on the
 * registration page before Auth0, that form opens; otherwise (login page → "Sign up",
 * Google, …) they choose here first: personal account or organisation.
 */
export function NewAccountRegistration({ registrationData }: { registrationData: RegistrationData }) {
  const { t } = useTranslation("auth");
  const navigate = useNavigate();
  const { refreshBackendUser, logout } = useAuth();
  // The stored choice is used once; it must not carry over to a later new account in this browser.
  const [choice, setChoice] = useState<RegistrationIntent | null>(() => getRegistrationIntent());
  useEffect(() => {
    clearRegistrationIntent();
  }, []);

  if (choice === null) {
    return (
      <Container size="xs" py="xl" maw={480}>
        <Card shadow="md" padding="xl" radius="md" withBorder>
          <Stack gap="lg">
            <Stack gap="xs" align="center" ta="center">
              <SectionTitle>{t("registerPage.title")}</SectionTitle>
              <BodyText size="sm">{t("registerPage.chooseType")}</BodyText>
            </Stack>
            <Stack gap="xs">
              <ActionButton leftSection={<IconUser size={20} />} onClick={() => setChoice("individual")} fullWidth>
                {t("registerPage.individualTitle")}
              </ActionButton>
              <Text size="xs" c="dimmed" ta="center">
                {t("registerPage.individualDescription")}
              </Text>
            </Stack>
            <Stack gap="xs">
              <ActionButton leftSection={<IconBuilding size={20} />} onClick={() => setChoice("organization")} fullWidth>
                {t("registerPage.organizationTitle")}
              </ActionButton>
              <Text size="xs" c="dimmed" ta="center">
                {t("registerPage.organizationIntro")}
              </Text>
            </Stack>
            <TextButton onClick={logout}>{t("register.cancelButton")}</TextButton>
          </Stack>
        </Card>
      </Container>
    );
  }

  if (choice === "individual") {
    return <RegistrationForm registrationData={registrationData} onCancel={() => setChoice(null)} />;
  }

  return (
    <OrganizationRegistrationForm
      mode="register"
      initialName={registrationData.name}
      initialEmail={registrationData.email}
      onSubmitted={async () => {
        // Account and organisation exist now (no e-mail confirmation for the time being).
        await refreshBackendUser();
        notifyOrganisationRegistered(t);
        navigate({ to: ROUTES.MY_ORGANIZATION as "/" });
      }}
      onCancel={() => setChoice(null)}
    />
  );
}
