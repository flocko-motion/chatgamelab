import { useCallback, useEffect, useState } from "react";
import { Alert, Card, Container, Group, Loader, Stack, Text } from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";
import { SectionTitle } from "@components/typography";
import type { ObjInstitutionApplication } from "@/api/generated";
import { useAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { useAuth } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { AgeGroup } from "@/constants/ageGroup";
import { OrganizationRegistrationForm } from "./OrganizationRegistrationForm";
import { PendingApplication } from "./PendingApplication";

/**
 * /organization/apply: shows the open application (check your mail), the upgrade form,
 * or why this account cannot become an organisation.
 */
export function OrganizationApplyPage() {
  const { t } = useTranslation("auth");
  const navigate = useNavigate();
  const api = useAuthenticatedApi();
  const { backendUser } = useAuth();
  const [pending, setPending] = useState<ObjInstitutionApplication | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);

  const load = useCallback(async () => {
    if (!api) return;
    try {
      const res = await api.institutionApplications.mineList();
      setPending(res.data.application ?? null);
    } catch {
      setPending(null);
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    load();
  }, [load]);

  if (loading || !backendUser) {
    return (
      <Group justify="center" py="xl">
        <Loader />
      </Group>
    );
  }

  const role = backendUser.role?.role;
  const isIndividual = !role || role === "individual";
  const isAdult = backendUser.ageGroup === AgeGroup.U18;

  if (role === "head" || role === "staff") {
    return (
      <Blocker
        title={t("organizationRegistration.blocked.memberTitle")}
        text={
          role === "head"
            ? t("organizationRegistration.blocked.head")
            : t("organizationRegistration.blocked.member")
        }
        action={role === "head" ? () => navigate({ to: ROUTES.MY_ORGANIZATION as "/" }) : undefined}
        actionLabel={t("organizationRegistration.toMyOrganization")}
      />
    );
  }
  if (!isIndividual) {
    return (
      <Blocker
        title={t("organizationRegistration.blocked.memberTitle")}
        text={t("organizationRegistration.blocked.member")}
      />
    );
  }
  if (!isAdult) {
    return (
      <Blocker
        title={t("organizationRegistration.blocked.ageTitle")}
        text={t("organizationRegistration.blocked.age")}
      />
    );
  }

  if (pending && !editing) {
    return <PendingApplication application={pending} onEdit={() => setEditing(true)} />;
  }

  return (
    <OrganizationRegistrationForm
      mode="upgrade"
      initialEmail={backendUser.email ?? ""}
      onSubmitted={() => {
        setEditing(false);
        setLoading(true);
        load();
      }}
      onCancel={() => (editing ? setEditing(false) : navigate({ to: ROUTES.SETTINGS as "/" }))}
    />
  );
}

function Blocker({
  title,
  text,
  action,
  actionLabel,
}: {
  title: string;
  text: string;
  action?: () => void;
  actionLabel?: string;
}) {
  return (
    <Container size="xs" py="xl">
      <Card shadow="md" padding="xl" radius="md" withBorder>
        <Stack gap="md">
          <SectionTitle>{title}</SectionTitle>
          <Alert color="blue" variant="light" icon={<IconInfoCircle size={16} />}>
            <Text size="sm">{text}</Text>
          </Alert>
          {action && actionLabel && <ActionButton onClick={action}>{actionLabel}</ActionButton>}
        </Stack>
      </Card>
    </Container>
  );
}
