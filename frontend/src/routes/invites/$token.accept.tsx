import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
  Container,
  Stack,
  Card,
  Center,
  Loader,
  Alert,
  Title,
  Text,
  Group,
  Image,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconCheck,
  IconArrowRight,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useState, useEffect, type ReactNode } from "react";
import { ActionButton } from "@/common/components/buttons/ActionButton";
import { TextButton } from "@/common/components/buttons/TextButton";
import { LanguageSwitcher } from "@/common/components/LanguageSwitcher";
import { acceptInvite, fetchInvite } from "@/api/hooks";
import { useAuth, rememberGuestCode } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { ErrorCodes } from "@/common/types/errorCodes";
import { buildShareUrl } from "@/common/lib/url";
import type { AcceptInviteResp, InviteCardView } from "@/api/types";
import { markWorkshopModeEntered } from "@/features/my-workshop/lib/infoModal";
import logo from "@/assets/logos/colorful/ChatGameLab-Logo-2025-Square-Colorful2-Black-Text.png-Black-Text-Transparent.png";

export const Route = createFileRoute("/invites/$token/accept")({
  component: AcceptInvitePage,
});

function InvitePageLayout({
  children,
  showBranding = true,
  showLanguageSwitcher = true,
}: {
  children: ReactNode;
  showBranding?: boolean;
  showLanguageSwitcher?: boolean;
}) {
  const { t } = useTranslation("common");
  return (
    <Container size="sm" py="xl">
      <Stack gap="xl" align="center">
        {showLanguageSwitcher && (
          <Group justify="flex-end" w="100%">
            <LanguageSwitcher size="sm" variant="subtle" />
          </Group>
        )}
        {children}
        {showBranding && (
          <Stack gap="sm" align="center" ta="center" mt="md">
            <Image
              src={logo}
              alt="ChatGameLab Logo"
              w={{ base: 200, sm: 280, lg: 350 }}
              h={{ base: 200, sm: 280, lg: 350 }}
              fit="contain"
            />
            <Text size="sm" c="dimmed" maw={400}>
              {t("home.splashDescription")}
            </Text>
          </Stack>
        )}
      </Stack>
    </Container>
  );
}

/** The status and error body of a failed API call; no status for a network failure. */
function failure(err: unknown): { status?: number; body: { code?: string; error?: string } } {
  const res = err as { status?: number; error?: { code?: string; error?: string } } | null;
  return { status: res?.status, body: res?.error ?? {} };
}

interface InviteDetails {
  id: string;
  workshopId?: string;
  workshopName?: string;
  institutionName?: string;
  role?: string;
  status?: string;
}

function AcceptInvitePage() {
  const { t } = useTranslation("common");
  const { token } = Route.useParams();
  const navigate = useNavigate();
  const {
    isGuest,
    isAuthenticated,
    backendUser,
    isLoading: authLoading,
  } = useAuth();
  // Check if user is a signed-in user with another role (admin, head, staff, individual)
  const isLoggedInNonGuest = isAuthenticated && !isGuest;

  const [state, setState] = useState<
    | "loading"
    | "ready"
    | "accepting"
    | "success"
    | "error"
    | "already-member"
  >("loading");
  const [invite, setInvite] = useState<InviteDetails | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isAccepting, setIsAccepting] = useState(false);

  // status >= 500 carries a raw English server sentence; never show that to a
  // Guest, who may be a teenager on a phone.
  const acceptErrorMessage = (
    errorData: { code?: string; error?: string },
    status?: number,
  ) => {
    if (errorData.code === ErrorCodes.TOKEN_LOCKED) {
      return t("invites.errors.tokenLocked");
    }
    if (errorData.code === ErrorCodes.GUEST_STAYS) {
      return t("invites.errors.guestStays");
    }
    if (status !== undefined && status >= 500) {
      return t("invites.errors.acceptFailed");
    }
    return errorData.error || t("invites.errors.acceptFailed");
  };

  // Get current workshop info if user is a Guest
  const currentWorkshopId = backendUser?.role?.workshop?.id;

  // Fetch invite details on mount
  useEffect(() => {
    // Wait for auth to finish loading
    if (authLoading) return;

    async function loadInvite() {
      try {
        let data: InviteCardView;
        try {
          data = await fetchInvite(token);
        } catch (err) {
          const { status, body } = failure(err);
          if (status === undefined) throw err;
          if (body.code === ErrorCodes.TOKEN_LOCKED) {
            setError(t("invites.errors.tokenLocked"));
          } else if (status === 404) {
            setError(t("invites.errors.notFound"));
          } else if (body.code === "invite_expired") {
            setError(t("invites.errors.expired"));
          } else {
            setError(t("invites.errors.loadFailed"));
          }
          setState("error");
          return;
        }
        setInvite(data);

        // Head/staff clicking a workshop invite link → auto-enter the workshop
        const userRole = backendUser?.role?.role;
        if (
          isLoggedInNonGuest &&
          data.workshopId &&
          (userRole === "head" || userRole === "staff")
        ) {
          // Auto-accept to enter workshop mode, then redirect
          try {
            await acceptInvite(token).catch(() => {});
            markWorkshopModeEntered();
            window.location.href = buildShareUrl(ROUTES.MY_WORKSHOP);
          } catch {
            // Fallback to showing the enter button
            setState("already-member");
          }
          return;
        }

        // Other signed-in users → offer to join; the server decides
        if (isLoggedInNonGuest) {
          setState("already-member");
          return;
        }

        // Check if user is already in this workshop
        if (
          isGuest &&
          currentWorkshopId &&
          data.workshopId === currentWorkshopId
        ) {
          // Already in this workshop - redirect directly
          navigate({ to: ROUTES.MY_WORKSHOP as "/" });
          return;
        }

        setState("ready");
      } catch {
        setError(t("invites.errors.loadFailed"));
        setState("error");
      }
    }

    loadInvite();
  }, [
    token,
    t,
    authLoading,
    isGuest,
    isLoggedInNonGuest,
    backendUser,
    currentWorkshopId,
    navigate,
  ]);

  const handleAccept = async () => {
    setIsAccepting(true);
    setState("accepting");

    try {
      // The session cookie tells the server who is asking, so a signed-in
      // Guest keeps their account instead of getting a second one.
      let result: AcceptInviteResp;
      try {
        result = await acceptInvite(token);
      } catch (err) {
        const { status, body } = failure(err);
        if (status === undefined) throw err;
        // Being recognised makes a new answer possible: "you are already in this
        // workshop". That is not a failure - send them where they wanted to go.
        if (body.code === ErrorCodes.CONFLICT) {
          navigate({ to: ROUTES.MY_WORKSHOP as "/" });
          return;
        }
        setError(acceptErrorMessage(body, status));
        setState("error");
        return;
      }

      // An anonymous join signs the new Guest in and returns their code;
      // keep it to show again. A recognised caller keeps the code they have.
      if (result.authToken) {
        rememberGuestCode(result.authToken);
      }

      setState("success");

      // Short delay to show success message, then redirect to my-workshop
      setTimeout(() => {
        // Force page reload to refresh auth state
        window.location.href = buildShareUrl(ROUTES.MY_WORKSHOP);
      }, 1500);
    } catch {
      setError(t("invites.errors.acceptFailed"));
      setState("error");
    }
  };

  if (state === "loading") {
    return (
      <InvitePageLayout>
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      </InvitePageLayout>
    );
  }

  if (state === "error") {
    return (
      <InvitePageLayout>
        <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
          <Stack align="center" gap="md">
            <Alert
              icon={<IconAlertCircle size={16} />}
              title={t("error")}
              color="red"
              w="100%"
            >
              {error}
            </Alert>
            <ActionButton onClick={() => navigate({ to: "/" })}>
              {t("invites.goHome")}
            </ActionButton>
          </Stack>
        </Card>
      </InvitePageLayout>
    );
  }

  if (state === "success") {
    return (
      <InvitePageLayout>
        <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
          <Stack align="center" gap="md">
            <IconCheck size={48} color="var(--mantine-color-green-6)" />
            <Title order={2} ta="center">
              {t("invites.success.title")}
            </Title>
            <Text ta="center" c="dimmed">
              {t("invites.success.message")}
            </Text>
            <Loader size="sm" />
          </Stack>
        </Card>
      </InvitePageLayout>
    );
  }

  // Show "enter workshop" option for authenticated users
  if (state === "already-member") {
    const handleEnterWorkshop = async () => {
      setIsAccepting(true);
      try {
        // Use the accept endpoint which handles all cases:
        // - Head/staff in same org → enters workshop mode
        // - Individual → takes part with their own role, in no organisation
        // - User without role → falls back to AcceptOpenInvite (joins as a Guest)
        try {
          await acceptInvite(token);
        } catch (err) {
          const { status, body } = failure(err);
          if (status === undefined) throw err;
          setError(acceptErrorMessage(body, status));
          setState("error");
          return;
        }

        // Force page reload to refresh auth state and workshop context
        markWorkshopModeEntered();
        window.location.href = buildShareUrl(ROUTES.MY_WORKSHOP);
      } catch {
        setError(t("invites.errors.acceptFailed"));
        setState("error");
      }
    };

    return (
      <InvitePageLayout showBranding={false} showLanguageSwitcher={false}>
        <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
          <Stack gap="lg">
            <Title order={2} ta="center">
              {t("invites.alreadyMember.title")}
            </Title>

            <Text ta="center" c="dimmed">
              {t("invites.alreadyMember.message", {
                workshop: invite?.workshopName,
              })}
            </Text>

            <Group justify="center" gap="md">
              <TextButton onClick={() => navigate({ to: "/" })}>
                {t("invites.goHome")}
              </TextButton>
              <ActionButton
                onClick={handleEnterWorkshop}
                loading={isAccepting}
                rightSection={<IconArrowRight size={16} />}
              >
                {t("invites.alreadyMember.enterButton")}
              </ActionButton>
            </Group>
          </Stack>
        </Card>
      </InvitePageLayout>
    );
  }

  return (
    <InvitePageLayout>
      <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
        <Stack gap="lg">
          <Title order={2} ta="center">
            {invite?.workshopName
              ? t("invites.accept.titleWithWorkshop", {
                  workshop: invite.workshopName,
                })
              : t("invites.accept.title")}
          </Title>

          <Text ta="center" c="dimmed">
            {invite?.institutionName
              ? t("invites.accept.inviteMessage", {
                  organization: invite.institutionName,
                })
              : t("invites.accept.inviteMessageSimple")}
          </Text>

          <ActionButton
            onClick={handleAccept}
            loading={state === "accepting"}
            fullWidth
          >
            {t("invites.accept.joinButton")}
          </ActionButton>
        </Stack>
      </Card>
    </InvitePageLayout>
  );
}
