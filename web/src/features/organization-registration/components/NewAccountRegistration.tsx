import { useState } from "react";
import { Alert, Anchor, Container, Text } from "@mantine/core";
import { IconBuilding } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { RegistrationForm } from "@/features/auth";
import { useAuth, type RegistrationData } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { OrganizationRegistrationForm } from "./OrganizationRegistrationForm";
import {
  clearRegistrationIntent,
  getRegistrationIntent,
  setRegistrationIntent,
} from "../lib/registrationIntent";
import { notifyOrganisationRegistered } from "../lib/notifyRegistered";

/**
 * Shown after the first login of a person without an account: the organisation form
 * if they chose "register as organisation" before the Auth0 signup, else the normal profile form.
 */
export function NewAccountRegistration({ registrationData }: { registrationData: RegistrationData }) {
  const { t } = useTranslation("auth");
  const navigate = useNavigate();
  const { refreshBackendUser } = useAuth();
  const [asOrganization, setAsOrganization] = useState(getRegistrationIntent() === "organization");

  if (!asOrganization) {
    // People can reach Auth0 without our selection page (login page "Sign up", Google):
    // offer the organisation route here too.
    return (
      <>
        <Container size="xs" pt={{ base: "md", sm: "xl" }}>
          <Alert color="blue" variant="light" icon={<IconBuilding size={18} />}>
            <Text size="sm">
              {t("organizationRegistration.switchHint")}{" "}
              <Anchor
                component="button"
                size="sm"
                fw={600}
                onClick={() => {
                  setRegistrationIntent("organization");
                  setAsOrganization(true);
                }}
              >
                {t("organizationRegistration.switchToOrganization")}
              </Anchor>
            </Text>
          </Alert>
        </Container>
        <RegistrationForm registrationData={registrationData} />
      </>
    );
  }

  return (
    <OrganizationRegistrationForm
      mode="register"
      initialName={registrationData.name}
      initialEmail={registrationData.email}
      onSubmitted={async () => {
        // Account and organisation exist now (no e-mail confirmation for the time being).
        clearRegistrationIntent();
        await refreshBackendUser();
        notifyOrganisationRegistered(t);
        navigate({ to: ROUTES.MY_ORGANIZATION as "/" });
      }}
      onCancel={() => {
        clearRegistrationIntent();
        setAsOrganization(false);
      }}
    />
  );
}
