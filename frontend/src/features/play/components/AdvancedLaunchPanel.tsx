import { Accordion, Badge, Center, Group, Loader, Paper, Stack, Textarea } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { PlayGameButton, TextButton } from "@components/buttons";
import { BodyText, CardTitle, HelperText } from "@components/typography";
import { useLaunchPrompts } from "../hooks/useLaunchPrompts";

interface AdvancedLaunchPanelProps {
  game: string;
  onStart: (prompts: Record<string, string>) => void;
  onBack?: () => void;
}

/**
 * The advanced settings before a session starts, for those who may adapt a
 * session's prompts: the genre's prompts, each open to a change, then the
 * start.
 */
export function AdvancedLaunchPanel({ game, onStart, onBack }: AdvancedLaunchPanelProps) {
  const { t } = useTranslation(["common", "playbackEngine"]);
  const launch = useLaunchPrompts(game);

  return (
    <Center style={{ flex: 1, alignItems: "flex-start" }} p="md">
      <Paper withBorder radius="md" p={{ base: "md", sm: "lg" }} shadow="sm" maw={760} w="100%">
        <Stack gap="md">
          <CardTitle>{t("play.advanced.title")}</CardTitle>
          <HelperText>{t("play.advanced.hint")}</HelperText>
          {launch.loading ? (
            <Group gap="sm" role="status">
              <Loader size="sm" />
              <HelperText>{t("play.advanced.loading")}</HelperText>
            </Group>
          ) : launch.failed ? (
            <BodyText>{t("play.advanced.unavailable")}</BodyText>
          ) : (
            <Accordion multiple variant="separated">
              {launch.prompts.map((prompt) => (
                <Accordion.Item key={prompt.name} value={prompt.name}>
                  <Accordion.Control>
                    <Group gap="xs">
                      {t(`promptNames.${prompt.name}`, { ns: "playbackEngine", defaultValue: prompt.name })}
                      {prompt.edited && (
                        <Badge size="sm" color="accent" variant="light">
                          {t("play.advanced.edited")}
                        </Badge>
                      )}
                    </Group>
                  </Accordion.Control>
                  <Accordion.Panel>
                    <Stack gap="xs">
                      <Textarea
                        label={t(`promptNames.${prompt.name}`, { ns: "playbackEngine", defaultValue: prompt.name })}
                        value={prompt.text}
                        onChange={(event) => launch.setText(prompt.name, event.currentTarget.value)}
                        autosize
                        minRows={4}
                        maxRows={18}
                        styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)" } }}
                      />
                      {prompt.edited && (
                        <TextButton size="xs" onClick={() => launch.reset(prompt.name)}>
                          {t("play.advanced.reset")}
                        </TextButton>
                      )}
                    </Stack>
                  </Accordion.Panel>
                </Accordion.Item>
              ))}
            </Accordion>
          )}
          <Group justify="space-between" wrap="wrap">
            {onBack ? <TextButton onClick={onBack}>{t("play.launch.backToGames")}</TextButton> : <span />}
            <PlayGameButton size="md" disabled={launch.loading} onClick={() => onStart(launch.overrides())}>
              {t("play.advanced.start")}
            </PlayGameButton>
          </Group>
        </Stack>
      </Paper>
    </Center>
  );
}
