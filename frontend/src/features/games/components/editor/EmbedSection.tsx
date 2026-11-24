import { Anchor, Code, CopyButton, Divider, Select, Stack, Switch } from "@mantine/core";
import { IconCheck, IconCopy } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { GenericButtonWithText } from "@components/buttons";
import { CardTitle, HelperText } from "@components/typography";
import { EMBED_STYLING_DOCS_URL } from "../../lib/embedSnippet";
import { useEmbedPanel } from "../../hooks/useEmbedPanel";

/**
 * Embedding, on the Sharing page: a toggle like "public" that shows the
 * panel when on. The organisation's verified page the game runs on, what
 * pays for it, and the snippet for the host page. Shown for a game of an
 * organisation, where the server lets the caller share it.
 */
export function EmbedSection({ game }: { game: GameView }) {
  const { t } = useTranslation("common");
  const panel = useEmbedPanel(game);
  if (!panel.shown) return null;

  return (
    <Stack gap="xs">
      <Divider />
      <Switch
        label={t("games.editor.embed.toggle")}
        checked={panel.on}
        onChange={(e) => panel.setOn(e.currentTarget.checked)}
        disabled={panel.loading || panel.saving}
      />
      <HelperText>{t("games.editor.embed.toggleHint")}</HelperText>

      {panel.on && (
        <Stack gap="md" mt="xs">
          {panel.pageOptions.length > 0 ? (
            <Select
              label={t("games.editor.embed.page")}
              description={t("games.editor.embed.pageHint")}
              data={panel.pageOptions}
              value={panel.pageId}
              onChange={panel.setPage}
              disabled={panel.saving}
              allowDeselect={false}
            />
          ) : (
            <HelperText c="orange.8">{t("games.editor.embed.noPages")}</HelperText>
          )}

          {panel.noKeys ? (
            <HelperText c="orange.8">{t("games.editor.embed.noKeys")}</HelperText>
          ) : (
            <Select
              label={t("games.editor.embed.paidBy")}
              placeholder={t("games.editor.embed.paidByPlaceholder")}
              data={panel.keyGroups}
              value={panel.shareId}
              onChange={panel.setShare}
              disabled={panel.saving}
              allowDeselect={false}
            />
          )}

          {panel.incomplete && <HelperText>{t("games.editor.embed.incomplete")}</HelperText>}

          {panel.snippet && (
            <Stack gap="xs">
              <CardTitle>{t("games.editor.embed.snippet")}</CardTitle>
              <HelperText>{t("games.editor.embed.snippetHint")}</HelperText>
              <HelperText>{t("games.editor.embed.snippetVerifiedHint")}</HelperText>
              <Code block>{panel.snippet}</Code>
              <CopyButton value={panel.snippet}>
                {({ copied, copy }) => (
                  <GenericButtonWithText
                    icon={copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                    variant="default"
                    size="sm"
                    onClick={copy}
                  >
                    {copied ? t("share.copied") : t("games.editor.embed.copy")}
                  </GenericButtonWithText>
                )}
              </CopyButton>
              <HelperText>{t("games.editor.embed.offNote")}</HelperText>
              <Anchor href={EMBED_STYLING_DOCS_URL} target="_blank" rel="noopener noreferrer" size="sm">
                {t("games.editor.embed.styleLink")}
              </Anchor>
            </Stack>
          )}
        </Stack>
      )}
    </Stack>
  );
}
