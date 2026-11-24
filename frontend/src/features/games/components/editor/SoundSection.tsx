import { useState } from "react";
import { Alert, Group, Input, Paper, SegmentedControl, Stack, Text, Textarea } from "@mantine/core";
import { IconAlertCircle, IconRefresh } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { DangerButton, GenericButtonWithText, TextButton } from "@components/buttons";
import { CardTitle, HelperText } from "@components/typography";
import { genreWording } from "../../lib/editorPages";
import { useGenreFields, useVoiceSlots, voiceChoice, withVoice } from "../../hooks/useGenreFields";
import type { GameEditor } from "../../hooks/useGameEditor";
import type { VoicePreview } from "../../hooks/useVoicePreview";
import { VoiceTile } from "./VoiceTile";

/**
 * Sound: how the game's voices sound, and which voice each role speaks in.
 * The engine names the voices a genre speaks in and the choices each has:
 * male, female or any have the session cast a fitting built-in voice; custom
 * makes a voice of its own from the voice style, which is the only voice
 * heard here, and remade, after asking first.
 */
export function SoundSection({ editor, preview }: { editor: GameEditor; preview: VoicePreview }) {
  const { t } = useTranslation("common");
  const { values, set, rights } = editor;
  const wording = genreWording(values.genre);
  const shows = useGenreFields(values.genre);
  const slots = useVoiceSlots(values.genre);
  const [confirming, setConfirming] = useState(false);
  const style = values.voiceStyle.trim();

  const status = preview.hasStyle ? t("games.editor.sound.status.ownVoice") : t("games.editor.sound.status.builtIn");

  return (
    <Stack gap="lg">
      {shows("voiceStyle") && (
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
      )}

      {slots.map((slot) => (
        <Input.Wrapper
          key={slot.name}
          label={t(`games.editor.sound.slots.${slot.name}`, {
            defaultValue: slot.name,
          })}
          description={
            slot.choices.includes("custom")
              ? `${t("games.editor.sound.choicesHint")} ${t("games.editor.sound.customHint")}`
              : t("games.editor.sound.choicesHint")
          }
        >
          <SegmentedControl
            mt={6}
            fullWidth
            value={voiceChoice(values.voices, slot)}
            onChange={(choice) => set("voices", withVoice(values.voices, slot, choice))}
            disabled={!rights.content}
            data={slot.choices.map((choice) => ({
              value: choice,
              label: t(`games.editor.sound.choices.${choice}`),
            }))}
          />
        </Input.Wrapper>
      ))}

      {!preview.custom && slots.length > 0 && <HelperText>{t("games.editor.sound.status.cast")}</HelperText>}

      {preview.custom && (
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
                {preview.playing && <HelperText>{t(`games.editor.sound.sampleLine.${preview.sampleLine}`)}</HelperText>}
                {rights.content && !confirming && (
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
            <HelperText>{t("games.editor.sound.noteDescription")}</HelperText>
            <HelperText>{t("games.editor.sound.noteKeys")}</HelperText>
          </Stack>
        </Paper>
      )}
    </Stack>
  );
}
