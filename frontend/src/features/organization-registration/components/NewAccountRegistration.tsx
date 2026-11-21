import { Card, Container, Stack } from "@mantine/core";
import { IconBuilding, IconUser } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { BodyText, HelperText, SectionTitle } from "@components/typography";
import { RegistrationForm } from "@/features/auth";
import { useAuth, type RegistrationData } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { OrganizationRegistrationForm } from "./OrganizationRegistrationForm";
import { useRegistrationChoice } from "../hooks/useRegistrationChoice";
import { notifyOrganisationRegistered } from "../lib/notifyRegistered";

/**
 * Shown after the first sign-in of a person without an account. The choice
 * made on the registration page opens its form; without one (signed up from
 * the login page, with Google, …) the person chooses here first.
 */
export function NewAccountRegistration({ registrationData }: { registrationData: RegistrationData }) {
  const { t } = useTranslation("auth");
  const navigate = useNavigate();
  const { logout } = useAuth();
  const [choice, setChoice] = useRegistrationChoice();

  if (choice === "individual") {
    return <RegistrationForm registrationData={registrationData} onCancel={() => setChoice(null)} />;
  }
  if (choice === "organization") {
    return (
      <OrganizationRegistrationForm
        initialName={registrationData.name}
        initialEmail={registrationData.email}
        onSubmitted={() => {
          notifyOrganisationRegistered(t);
          navigate({ to: ROUTES.MY_ORGANIZATION });
        }}
        onCancel={() => setChoice(null)}
      />
    );
  }

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
            <HelperText ta="center">{t("registerPage.individualDescription")}</HelperText>
          </Stack>
          <Stack gap="xs">
            <ActionButton leftSection={<IconBuilding size={20} />} onClick={() => setChoice("organization")} fullWidth>
              {t("registerPage.organizationTitle")}
            </ActionButton>
            <HelperText ta="center">{t("registerPage.organizationIntro")}</HelperText>
          </Stack>
          <TextButton onClick={logout}>{t("register.cancelButton")}</TextButton>
        </Stack>
      </Card>
    </Container>
  );
}
