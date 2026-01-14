import { createFileRoute, useRouter } from "@tanstack/react-router";
import {
  Text,
  Stack,
  Center,
  Image,
  Container,
  Title,
  SimpleGrid,
  Card,
  Group,
  ThemeIcon,
  Box,
  useMantineTheme,
} from "@mantine/core";
import { useTranslation } from "react-i18next";
import { Button } from "@components/Button";
import { LanguageSwitcher } from "@components/LanguageSwitcher";
import {
  IconBook,
  IconUsers,
  IconSchool,
  IconSparkles,
  IconRocket,
} from "@tabler/icons-react";
import logo from "@/assets/logos/colorful/ChatGameLab-Logo-2025-Square-Colorful2-Black-Text.png-Black-Text-Transparent.png";
import { ROUTES } from "@/common/routes/routes";

export const Route = createFileRoute(ROUTES.HOME)({
  component: HomePage,
});

function HomePage() {
  const { t } = useTranslation("common");
  const router = useRouter();
  const theme = useMantineTheme();

  const features = [
    {
      icon: IconBook,
      title: t("home.features.create.title", "Create Adventures"),
      description: t(
        "home.features.create.description",
        "Design interactive text adventures powered by AI",
      ),
    },
    {
      icon: IconSparkles,
      title: t("home.features.understand.title", "Understand AI"),
      description: t(
        "home.features.understand.description",
        "Explore how AI models create stories and learn how they work",
      ),
    },
    {
      icon: IconSchool,
      title: t("home.features.learn.title", "Learn & Teach"),
      description: t(
        "home.features.learn.description",
        "Perfect for educational workshops and classroom activities",
      ),
    },
    {
      icon: IconUsers,
      title: t("home.features.play.title", "Play Together"),
      description: t(
        "home.features.play.description",
        "Join friends in collaborative storytelling sessions",
      ),
    },
        {
      icon: IconUsers,
      title: t("home.features.play.title", "Play Together"),
      description: t(
        "home.features.play.description",
        "Join friends in collaborative storytelling sessions",
      ),
    },
        {
      icon: IconUsers,
      title: t("home.features.play.title", "Play Together"),
      description: t(
        "home.features.play.description",
        "Join friends in collaborative storytelling sessions",
      ),
    },
        {
      icon: IconUsers,
      title: t("home.features.play.title", "Play Together"),
      description: t(
        "home.features.play.description",
        "Join friends in collaborative storytelling sessions",
      ),
    }
    
  ];

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
                alt="ChatGameLab Logo"
                w={{ base: 200, sm: 280, lg: 350 }}
                h={{ base: 200, sm: 280, lg: 350 }}
                fit="contain"
              />

              <Text size="xl" c="gray.8" maw={600} lh={1.6}>
                {t(
                  "home.splashDescription",
                  "An educational platform for creating and playing AI-powered text-adventure games. Perfect for teachers, students, and creative storytellers.",
                )}
              </Text>

              <Button
                variant="primary"
                color="accent"
                textColor="#083344"
                size="lg"
                onClick={() => {
                  router.navigate({ to: ROUTES.AUTH_LOGIN });
                }}
                leftSection={<IconRocket size={20} />}
                fullWidth={true}
                w={{ base: "100%", sm: "auto" }}
                miw={200}
                mt="lg"
              >
                {t("home.loginCta", "Get Started")}
              </Button>
            </Stack>
          </Center>

          {/* Features Section */}
          <Stack gap="xl" mt="xl">
            <Stack gap="md" align="center" ta="center">
              <Title order={2} size="h2" c="accent.9.5">
                {t("home.features.title", "Why Choose ChatGameLab?")}
              </Title>
            </Stack>

            <SimpleGrid
              cols={{ base: 1, sm: 2, lg: 4 }}
              spacing={{ base: "lg", sm: "xl" }}
              verticalSpacing={{ base: "lg", sm: "xl" }}
            >
              {features.map((feature, index) => (
                <Card
                  key={index}
                  shadow="md"
                  p="lg"
                  radius="md"
                  withBorder={false}
                  h="100%"
                  bg={theme.other.colors.bgCard}
                  style={{
                    transition: "all 0.3s ease",
                    border: `1px solid ${theme.other.colors.bgCardBorder}`,
                  }}
                >
                  <Stack gap="md" align="center" ta="center">
                    <ThemeIcon
                      size="xl"
                      radius="xl"
                      color="accent"
                      variant="light"
                    >
                      <feature.icon size={24} />
                    </ThemeIcon>

                    <Title order={3} size="h4" c="accent.9.5">
                      {feature.title}
                    </Title>

                    <Text size="sm" c="gray.7" lh={1.5}>
                      {feature.description}
                    </Text>
                  </Stack>
                </Card>
              ))}
            </SimpleGrid>
          </Stack>
        </Stack>
      </Container>
    </Box>
  );
}
