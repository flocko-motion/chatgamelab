import { createFileRoute, useRouter } from "@tanstack/react-router";
import {
  Container,
  Paper,
  Stack,
  Checkbox,
  Anchor,
  Divider,
  Group,
  List,
  Button as MantineButton,
} from "@mantine/core";
import { ActionButton } from "@components/buttons";
import { SectionTitle, BodyText, HelperText } from "@components/typography";
import { useAuth } from "@/providers/AuthProvider";
import { useTranslation } from "react-i18next";
import { useEffect, useState } from "react";
import {
  IconUser,
  IconBuilding,
  IconShieldCheck,
  IconArrowLeft,
} from "@tabler/icons-react";
import { ROUTES } from "@/common/routes/routes";
import { HELP_LINKS } from "@/config/helpLinks";
import { LanguageSwitcher } from "@components/LanguageSwitcher";
import {
  setRegistrationIntent,
  clearRegistrationIntent,
} from "@/features/organization-registration";

export const Route = createFileRoute("/auth/register/")({
  component: RegisterComponent,
});

type RegisterStep = "choose-type" | "individual-register" | "organization-register";

function RegisterComponent() {
  const { t } = useTranslation("auth");
  const { loginWithAuth0, loginWithRole, loginAsNewPerson, isDevMode, user } = useAuth();
  const router = useRouter();
  const [step, setStep] = useState<RegisterStep>("choose-type");
  const [agreedToTerms, setAgreedToTerms] = useState(false);

  // Redirect authenticated users — routing hub decides where to go
  useEffect(() => {
    if (user) {
      router.navigate({ to: ROUTES.HOME });
    }
  }, [user, router]);

  const handleRegister = () => {
    setRegistrationIntent("individual");
    loginWithAuth0({ screenHint: "signup" });
  };

  // Organisations: after the Auth0 signup the app shows the organisation form.
  const handleOrganizationRegister = () => {
    setRegistrationIntent("organization");
    loginWithAuth0({ screenHint: "signup" });
  };

  // Dev roles matching backend preseed users
  const devRoles = [
    { key: "admin-1", label: "Admin 1", color: "red" },
    { key: "admin-2", label: "Admin 2", color: "red" },
    { key: "head-1", label: "Head 1 (Orga)", color: "violet" },
    { key: "head-2", label: "Head 2 (Orga)", color: "violet" },
    { key: "staff-1", label: "Staff 1 (Orga)", color: "blue" },
    { key: "staff-2", label: "Staff 2 (Orga)", color: "blue" },
    { key: "individual-1", label: "Individual 1", color: "gray" },
    { key: "individual-2", label: "Individual 2", color: "gray" },
    { key: "participant", label: "Participant (Workshop)", color: "teal" },
  ];

  return (
    <Container size="xs" py="xl" maw={480}>
      <Group justify="flex-end" mb="md">
        <LanguageSwitcher size="sm" variant="subtle" />
      </Group>

      {/* Step 1: Choose registration type */}
      {step === "choose-type" && (
        <Paper shadow="md" p="xl" withBorder>
          <Stack gap="lg">
            <Stack gap="xs" align="center" ta="center">
              <SectionTitle>
                {t("registerPage.title", "Create Your Account")}
              </SectionTitle>
              <BodyText size="sm">
                {t(
                  "registerPage.chooseType",
                  "How would you like to use ChatGameLab?",
                )}
              </BodyText>
            </Stack>

            <Stack gap="md">
              <ActionButton
                leftSection={<IconUser size={20} />}
                onClick={() => setStep("individual-register")}
                fullWidth
              >
                {t("registerPage.individualTitle", "Individual User")}
              </ActionButton>

              <ActionButton
                leftSection={<IconBuilding size={20} />}
                onClick={() => setStep("organization-register")}
                fullWidth
              >
                {t("registerPage.organizationTitle")}
              </ActionButton>
            </Stack>

            <HelperText ta="center">
              {t("registerPage.alreadyHaveAccount", "Already have an account?")}{" "}
              <Anchor
                component="button"
                size="sm"
                onClick={() => router.navigate({ to: ROUTES.AUTH_LOGIN })}
              >
                {t("registerPage.loginLink", "Log in")}
              </Anchor>
            </HelperText>
          </Stack>
        </Paper>
      )}

      {/* Step 2: Individual registration (Auth0 + terms) */}
      {step === "individual-register" && (
        <Paper shadow="md" p="xl" withBorder>
          <Stack gap="lg">
            <Stack gap="xs" align="center" ta="center">
              <IconShieldCheck
                size={48}
                stroke={1.5}
                color="var(--mantine-color-blue-6)"
              />
              <SectionTitle>
                {t("registerPage.title", "Create Your Account")}
              </SectionTitle>
              <BodyText size="sm">
                {t(
                  "registerPage.description",
                  "We use Auth0 as our authentication provider to manage user accounts. You will be redirected to Auth0 to create your account, then returned here to complete your profile.",
                )}
              </BodyText>
            </Stack>

            <Divider />

            <Checkbox
              label={
                <BodyText size="sm" component="span">
                  {t("registerPage.agreePrefix", "I agree to the")}{" "}
                  <Anchor
                    href={HELP_LINKS.TERMS_OF_SERVICE}
                    target="_blank"
                    rel="noopener noreferrer"
                    size="sm"
                  >
                    {t("registerPage.termsOfService", "Terms of Service")}
                  </Anchor>{" "}
                  {t("registerPage.agreeSuffix", "and")}{" "}
                  <Anchor
                    href={HELP_LINKS.PRIVACY_POLICY}
                    target="_blank"
                    rel="noopener noreferrer"
                    size="sm"
                  >
                    {t("registerPage.privacyPolicy", "Privacy Policy")}
                  </Anchor>
                  {t("registerPage.agreeSuffix2") && ` ${t("registerPage.agreeSuffix2")}`}
                </BodyText>
              }
              checked={agreedToTerms}
              onChange={(event) =>
                setAgreedToTerms(event.currentTarget.checked)
              }
            />

            <ActionButton
              onClick={handleRegister}
              fullWidth
              disabled={!agreedToTerms}
            >
              {t("registerPage.continueButton", "Continue to Registration")}
            </ActionButton>

            <ActionButton
              onClick={() => {
                setStep("choose-type");
                setAgreedToTerms(false);
              }}
              leftSection={<IconArrowLeft size={16} />}
              color="gray"
              fullWidth
            >
              {t("registerPage.backButton", "Back")}
            </ActionButton>
          </Stack>
        </Paper>
      )}

      {/* Step 2: Organisation registration (Auth0, then the organisation form) */}
      {step === "organization-register" && (
        <Paper shadow="md" p="xl" withBorder>
          <Stack gap="lg">
            <Stack gap="xs" align="center" ta="center">
              <IconBuilding
                size={48}
                stroke={1.5}
                color="var(--mantine-color-blue-6)"
              />
              <SectionTitle>{t("registerPage.organizationTitle")}</SectionTitle>
              <BodyText size="sm">{t("registerPage.organizationIntro")}</BodyText>
            </Stack>

            <List size="sm" spacing="xs" type="ordered">
              <List.Item>{t("registerPage.organizationStep1")}</List.Item>
              <List.Item>{t("registerPage.organizationStep2")}</List.Item>
              <List.Item>{t("registerPage.organizationStep3")}</List.Item>
            </List>

            <HelperText>
              {t("registerPage.organizationExistingAccount")}{" "}
              <Anchor
                component="button"
                size="sm"
                onClick={() => router.navigate({ to: ROUTES.AUTH_LOGIN })}
              >
                {t("registerPage.loginLink", "Log in")}
              </Anchor>
            </HelperText>

            <ActionButton onClick={handleOrganizationRegister} fullWidth>
              {t("registerPage.continueButton", "Continue to Registration")}
            </ActionButton>

            <ActionButton
              onClick={() => setStep("choose-type")}
              leftSection={<IconArrowLeft size={16} />}
              color="gray"
              fullWidth
            >
              {t("registerPage.backButton", "Back")}
            </ActionButton>
          </Stack>
        </Paper>
      )}

      {/* Dev Mode Quick Login */}
      {isDevMode && (
        <Paper shadow="md" p="xl" withBorder mt="lg">
          <Stack gap="md">
            <Divider
              label={t("login.devMode")}
              labelPosition="center"
            />
            <Stack gap="sm">
              <HelperText>{t("login.devModeDescription")}</HelperText>
              <MantineButton
                variant="outline"
                color="orange"
                onClick={async () => {
                  setRegistrationIntent("organization");
                  await loginAsNewPerson();
                }}
                fullWidth
              >
                Neue Person (Organisation registrieren)
              </MantineButton>
              <MantineButton
                variant="outline"
                color="orange"
                onClick={async () => {
                  clearRegistrationIntent();
                  await loginAsNewPerson();
                }}
                fullWidth
              >
                Neue Person (ohne Vorauswahl, wie „Anmelden“ → „Sign up“)
              </MantineButton>
              {devRoles.map((role) => (
                <MantineButton
                  key={role.key}
                  variant={role.key === "admin-1" ? "filled" : "outline"}
                  color={role.color}
                  onClick={async () => {
                    await loginWithRole(role.key);
                    router.navigate({ to: ROUTES.DASHBOARD });
                  }}
                  fullWidth
                >
                  {role.label}
                </MantineButton>
              ))}
            </Stack>
          </Stack>
        </Paper>
      )}
    </Container>
  );
}
