import { createFileRoute, useRouter } from "@tanstack/react-router";
import {
  Container,
  Paper,
  Stack,
  Checkbox,
  Anchor,
  Divider,
  Modal,
  CopyButton,
  ActionIcon,
  Tooltip,
  Group,
  Button as MantineButton,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { ActionButton } from "@components/buttons";
import { SectionTitle, BodyText, HelperText } from "@components/typography";
import { useAuth } from "@/providers/AuthProvider";
import { useTranslation } from "react-i18next";
import { useEffect, useState } from "react";
import {
  IconUser,
  IconSchool,
  IconShieldCheck,
  IconArrowLeft,
  IconCheck,
  IconCopy,
} from "@tabler/icons-react";
import { ROUTES } from "@/common/routes/routes";
import { HELP_LINKS, getHelpUrl, CONTACT_EMAILS } from "@/config/helpLinks";
import { LanguageSwitcher } from "@components/LanguageSwitcher";

export const Route = createFileRoute("/auth/register/")({
  component: RegisterComponent,
});

type RegisterStep = "choose-type" | "individual-register";

function RegisterComponent() {
  const { t } = useTranslation("auth");
  const { loginWithAuth0, loginWithRole, isDevMode, user } = useAuth();
  const router = useRouter();
  const [step, setStep] = useState<RegisterStep>("choose-type");
  const [agreedToTerms, setAgreedToTerms] = useState(false);
  const [educatorModalOpened, { open: openEducatorModal, close: closeEducatorModal }] =
    useDisclosure(false);

  // Redirect to dashboard if already authenticated
  useEffect(() => {
    if (user) {
      router.navigate({ to: ROUTES.DASHBOARD });
    }
  }, [user, router]);

  const handleRegister = () => {
    loginWithAuth0();
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
                leftSection={<IconSchool size={20} />}
                onClick={openEducatorModal}
                fullWidth
              >
                {t("registerPage.educatorTitle", "Educator / Professional")}
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
                    href={getHelpUrl(HELP_LINKS.TERMS_OF_SERVICE)}
                    target="_blank"
                    rel="noopener noreferrer"
                    size="sm"
                  >
                    {t("registerPage.termsOfService", "Terms of Service")}
                  </Anchor>
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

      {/* Educator / Professional Modal */}
      <Modal
        opened={educatorModalOpened}
        onClose={closeEducatorModal}
        title={
          <SectionTitle>
            {t(
              "registerPage.educatorModal.title",
              "Educator & Professional Access",
            )}
          </SectionTitle>
        }
        size="md"
        centered
      >
        <Stack gap="md">
          <BodyText size="sm">
            {t(
              "registerPage.educatorModal.intro",
              "To use ChatGameLab with your organization (school, university, institution), please follow these steps:",
            )}
          </BodyText>

          <Stack gap="sm">
            <BodyText size="sm" fw={600}>
              {t("registerPage.educatorModal.step1", "1. Register as an individual user first")}
            </BodyText>
            <BodyText size="sm">
              {t(
                "registerPage.educatorModal.step1Description",
                "Create your personal account using the individual registration. This will be your login for the platform.",
              )}
            </BodyText>

            <BodyText size="sm" fw={600}>
              {t("registerPage.educatorModal.step2", "2. Send us an email")}
            </BodyText>
            <BodyText size="sm">
              {t(
                "registerPage.educatorModal.step2Description",
                "Write an email to the address below with:",
              )}
            </BodyText>
            <Stack gap={4} pl="md">
              <BodyText size="sm">
                {t(
                  "registerPage.educatorModal.step2Item1",
                  "Your registered email address",
                )}
              </BodyText>
              <BodyText size="sm">
                {t(
                  "registerPage.educatorModal.step2Item2",
                  "The name of your organization (school, institution, etc.)",
                )}
              </BodyText>
            </Stack>

            <Group gap="xs" align="center">
              <Anchor
                href={`mailto:${CONTACT_EMAILS.ORGANIZATION_REQUEST}`}
                size="sm"
                fw={600}
              >
                {CONTACT_EMAILS.ORGANIZATION_REQUEST}
              </Anchor>
              <CopyButton value={CONTACT_EMAILS.ORGANIZATION_REQUEST}>
                {({ copied, copy }) => (
                  <Tooltip
                    label={copied ? t("registerPage.educatorModal.copied", "Copied!") : t("registerPage.educatorModal.copyEmail", "Copy email")}
                    withArrow
                  >
                    <ActionIcon
                      variant="subtle"
                      color={copied ? "teal" : "gray"}
                      onClick={copy}
                      size="sm"
                    >
                      {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
                    </ActionIcon>
                  </Tooltip>
                )}
              </CopyButton>
            </Group>

            <BodyText size="sm" fw={600}>
              {t(
                "registerPage.educatorModal.step3",
                "3. We'll set up your organization",
              )}
            </BodyText>
            <BodyText size="sm">
              {t(
                "registerPage.educatorModal.step3Description",
                "We will create your organization and assign you as the head. You can then invite colleagues and set up workshops.",
              )}
            </BodyText>
          </Stack>

          <Divider />

          <Group justify="space-between">
            <ActionButton onClick={closeEducatorModal} color="gray">
              {t("registerPage.educatorModal.closeButton", "Close")}
            </ActionButton>
            <ActionButton
              onClick={() => {
                closeEducatorModal();
                setStep("individual-register");
              }}
            >
              {t(
                "registerPage.educatorModal.registerButton",
                "Register as Individual",
              )}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}
