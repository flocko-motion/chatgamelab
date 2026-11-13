import { useEffect, useRef, useState } from "react";
import { Alert, Card, Container, Group, Loader, Stack, Text } from "@mantine/core";
import { IconAlertCircle, IconCircleCheck } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";
import { SectionTitle } from "@components/typography";
import { Api } from "@/api/generated";
import { getApiConfig } from "@/api/client/http";
import { useAuth } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { organizationErrorMessage } from "../lib/errors";
import { clearRegistrationIntent } from "../lib/registrationIntent";

/** /organization/confirm?token=…: the link from the e-mail. Works with or without login. */
export function OrganizationConfirmPage({ token }: { token: string }) {
  const { t } = useTranslation("auth");
  const navigate = useNavigate();
  const { isAuthenticated, retryBackendFetch } = useAuth();
  const [state, setState] = useState<"working" | "done" | "error">("working");
  const [institutionName, setInstitutionName] = useState("");
  const [error, setError] = useState("");
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    new Api(getApiConfig()).institutionApplications
      .confirmCreate({ token })
      .then((res) => {
        setInstitutionName(res.data.institution?.name ?? "");
        setState("done");
        clearRegistrationIntent();
        // The account just became head: reload it so menus show the organisation.
        retryBackendFetch();
      })
      .catch((e) => {
        setError(organizationErrorMessage(e, t));
        setState("error");
      });
  }, [token, t, retryBackendFetch]);

  return (
    <Container size="xs" py="xl">
      <Card shadow="md" padding="xl" radius="md" withBorder>
        {state === "working" && (
          <Group justify="center" py="lg">
            <Loader />
          </Group>
        )}
        {state === "done" && (
          <Stack gap="md" align="center" ta="center">
            <IconCircleCheck size={56} stroke={1.5} color="var(--mantine-color-green-6)" />
            <SectionTitle>{t("organizationRegistration.confirm.successTitle")}</SectionTitle>
            <Text size="sm">
              {t("organizationRegistration.confirm.successText", { name: institutionName })}
            </Text>
            {isAuthenticated ? (
              <ActionButton onClick={() => navigate({ to: ROUTES.MY_ORGANIZATION as "/" })}>
                {t("organizationRegistration.toMyOrganization")}
              </ActionButton>
            ) : (
              <ActionButton onClick={() => navigate({ to: ROUTES.AUTH_LOGIN as "/" })}>
                {t("organizationRegistration.confirm.login")}
              </ActionButton>
            )}
          </Stack>
        )}
        {state === "error" && (
          <Stack gap="md">
            <SectionTitle>{t("organizationRegistration.confirm.errorTitle")}</SectionTitle>
            <Alert color="red" icon={<IconAlertCircle size={16} />}>
              {error}
            </Alert>
            {isAuthenticated && (
              <ActionButton onClick={() => navigate({ to: ROUTES.ORGANIZATION_APPLY as "/" })}>
                {t("organizationRegistration.confirm.backToApplication")}
              </ActionButton>
            )}
          </Stack>
        )}
      </Card>
    </Container>
  );
}
