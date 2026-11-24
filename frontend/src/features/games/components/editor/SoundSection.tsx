import { useState } from "react";
import { Alert, Group, Paper, Stack, Text, Textarea } from "@mantine/core";
import { IconAlertCircle, IconRefresh } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { DangerButton, GenericButtonWithText, TextButton } from "@components/buttons";
import { CardTitle, HelperText } from "@components/typography";
import { genreWording } from "../../lib/editorPages";
import type { GameEditor } from "../../hooks/useGameEditor";
import type { VoicePreview } from "../../hooks/useVoicePreview";
import { VoiceTile } from "./VoiceTile";

/**
 * Sound: how the game's voice sounds, and the voice itself to listen to. The
 * same for every genre; only the words follow it (the narrator's voice, the
 * character's). Making a new voice is offered where the genre makes voices
 * of its own, and asks first.
 */
export function SoundSection({ editor, preview }: { editor: GameEditor; preview: VoicePreview }) {
  const { t } = useTranslation("common");
  const { values, set, rights } = editor;
  const wording = genreWording(values.genre);
  const [confirming, setConfirming] = useState(false);
  const style = values.voiceStyle.trim();

  let status = t("games.editor.sound.status.builtIn");
  if (preview.hasStyle) {
    status = preview.makesVoices
      ? t("games.editor.sound.status.ownVoice")
      : t("games.editor.sound.status.steered");
  }

  return (
    <Stack gap="lg">
      <Textarea
        label={t(`games.editor.sound.${wording}.voiceStyle`)}
        description={t(`games.editor.sound.${wording}.voiceStyleHint`)}
        placeholder={t("games.editFields.voiceStylePlaceholder")}
        value={values.voiceStyle}
        onChange={(e) => set("voiceStyle", e.target.value)}
        minRows={2}
        autosize
        readOnly={!rights.content}
      />

      <Paper withBorder radius="md" p="md">
        <Stack gap="sm">
          {/* Laid out like the cover: the tile, and beside it what to do. */}
          <Group align="flex-start" gap="md" wrap="wrap">
            <div style={{ flex: "1 1 220px", maxWidth: 360 }}>
              <VoiceTile preview={preview} style={style} />
            </div>
            <Stack gap="xs" style={{ flex: "1 1 260px" }}>
              <CardTitle>{t("games.editor.sound.listen")}</CardTitle>
              <HelperText>{status}</HelperText>
              {preview.playing && (
                <HelperText>{t(`games.editor.sound.sampleLine.${preview.sampleLine}`)}</HelperText>
              )}
              {preview.makesVoices && rights.content && !confirming && (
                <Group>
                  <GenericButtonWithText
                    icon={<IconRefresh size={16} />}
                    variant="default"
                    size="sm"
                    onClick={() => setConfirming(true)}
                    disabled={!preview.mayRenew || preview.renewing || preview.loading}
                    loading={preview.renewing}
                  >
                    {t("games.editor.sound.newVoice")}
                  </GenericButtonWithText>
                </Group>
              )}
            </Stack>
          </Group>

          {confirming && (
            <Alert color="orange" variant="light">
              <Stack gap="xs">
                <Text size="sm">{t("games.editor.sound.newVoiceConfirm")}</Text>
                <Group gap="xs">
                  <DangerButton
                    onClick={() => {
                      setConfirming(false);
                      void preview.renew();
                    }}
                  >
                    {t("games.editor.sound.newVoice")}
                  </DangerButton>
                  <TextButton onClick={() => setConfirming(false)}>{t("games.editor.sound.keepVoice")}</TextButton>
                </Group>
              </Stack>
            </Alert>
          )}

          {preview.errorKey && (
            <Alert icon={<IconAlertCircle size={16} />} color="orange" variant="light">
              {t(preview.errorKey)}
            </Alert>
          )}

          {/* Both notes are about a voice of its own, kept per key. */}
          {preview.makesVoices && (
            <>
              <HelperText>{t("games.editor.sound.noteDescription")}</HelperText>
              <HelperText>{t("games.editor.sound.noteKeys")}</HelperText>
            </>
          )}
        </Stack>
      </Paper>
    </Stack>
  );
}
