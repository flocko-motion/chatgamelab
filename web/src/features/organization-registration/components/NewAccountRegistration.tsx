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
        // The account exists now; the organisation follows once the e-mail is confirmed.
        await refreshBackendUser();
        navigate({ to: ROUTES.ORGANIZATION_APPLY as "/" });
      }}
      onCancel={() => {
        clearRegistrationIntent();
        setAsOrganization(false);
      }}
    />
  );
}
