import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { RegistrationForm } from "@/features/auth";
import { useAuth, type RegistrationData } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { OrganizationRegistrationForm } from "./OrganizationRegistrationForm";
import { clearRegistrationIntent, getRegistrationIntent } from "../lib/registrationIntent";

/**
 * Shown after the first login of a person without an account: the organisation form
 * if they chose "register as organisation" before the Auth0 signup, else the normal profile form.
 */
export function NewAccountRegistration({ registrationData }: { registrationData: RegistrationData }) {
  const navigate = useNavigate();
  const { refreshBackendUser } = useAuth();
  const [asOrganization, setAsOrganization] = useState(getRegistrationIntent() === "organization");

  if (!asOrganization) {
    return <RegistrationForm registrationData={registrationData} />;
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
