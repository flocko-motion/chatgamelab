import { createFileRoute, useRouter } from "@tanstack/react-router";
import { Stack, Center, Image, Container, Group, Box } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { useEffect } from "react";
import { ActionButton } from "@components/buttons";
import { BodyText } from "@components/typography";
import { LanguageSwitcher } from "@components/LanguageSwitcher";
import { IconRocket, IconLogin, IconKeyboard } from "@tabler/icons-react";
import logo from "@/assets/logos/colorful/ChatGameLab-Logo-2025-Square-Colorful2-Black-Text.png-Black-Text-Transparent.png";
import { ROUTES } from "@/common/routes/routes";
import { hasExternalHomepage, getHomepageUrl } from "@/common/lib/url";
import { useAuth } from "@/providers/AuthProvider";
import { useBranding } from "@/providers/branding";

export const Route = createFileRoute("/landing")({
  component: LandingPage,
});

function LandingPage() {
  const { t } = useTranslation("common");
  const branding = useBranding();
  const router = useRouter();
  const { user } = useAuth();

  // If external homepage is configured, this page should never be shown — redirect
  useEffect(() => {
    if (hasExternalHomepage()) {
      window.location.href = getHomepageUrl();
    }
  }, []);

  // Redirect to dashboard if already authenticated
  useEffect(() => {
    if (user) {
      router.navigate({ to: ROUTES.DASHBOARD });
    }
  }, [user, router]);

  return (
    <Box>
      {/* Hero Section */}
      <Container size="xl" px={{ base: "sm", sm: "md", lg: "xl" }}>
        <Stack gap="xl" py="md" mt="-sm">
          <Group justify="space-between" align="center">
            <Box />
            <LanguageSwitcher size="sm" variant="subtle" />
          </Group>

          <Center>
            <Stack gap="xl" align="center" ta="center" maw={800}>
              <Image
                src={logo}
                alt={branding.siteName}
                w={{ base: 200, sm: 280, lg: 350 }}
                h={{ base: 200, sm: 280, lg: 350 }}
                fit="contain"
              />

              <BodyText size="xl">
                {t(
                  "home.splashDescription",
                  "An educational platform for creating and playing AI-powered text-adventure games. Perfect for teachers, students, and creative storytellers.",
                )}
              </BodyText>

              <Group gap="md" justify="center">
                <ActionButton
                  onClick={() => {
                    router.navigate({ to: ROUTES.AUTH_REGISTER });
                  }}
                  leftSection={<IconRocket size={20} />}
                  size="md"
                >
                  {t("home.registerCta", "Register")}
                </ActionButton>
                <ActionButton
                  onClick={() => {
                    router.navigate({ to: ROUTES.AUTH_LOGIN });
                  }}
                  leftSection={<IconLogin size={20} />}
                  size="md"
                  color="gray"
                >
                  {t("home.loginCta", "Log in")}
                </ActionButton>
                <ActionButton
                  onClick={() => {
                    router.navigate({ to: ROUTES.CODE });
                  }}
                  leftSection={<IconKeyboard size={20} />}
                  size="md"
                  color="gray"
                >
                  {t("home.codeCta")}
                </ActionButton>
              </Group>
            </Stack>
          </Center>

        </Stack>
      </Container>
    </Box>
  );
}
