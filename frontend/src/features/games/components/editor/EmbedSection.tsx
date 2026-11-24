import { Anchor, Code, CopyButton, Select, Stack, Switch } from "@mantine/core";
import { IconCheck, IconCopy } from "@tabler/icons-react";
import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { GenericButtonWithText } from "@components/buttons";
import { BodyText, HelperText, Label } from "@components/typography";
import { EMBED_STYLING_DOCS_URL } from "../../lib/embedSnippet";
import type { EmbedPanel } from "../../hooks/useEmbedPanel";
import { SharingBlock } from "./SharingBlock";

/**
 * Embedding, on the Sharing page: a block with a toggle like "public" that
 * shows the panel when on. The organisation's verified page the game runs
 * on, what pays for it, and the snippet for the host page. The Sharing page
 * shows it where the panel is `shown` (the server lets the caller share the
 * game); while embedding cannot work yet, it says what is to be done first,
 * in order, and the toggle stays off.
 */
export function EmbedSection({ panel }: { panel: EmbedPanel }) {
  const { t } = useTranslation("common");

  return (
    <SharingBlock
      title={t("games.editor.embed.toggle")}
      description={t("games.editor.embed.toggleHint")}
      control={
        <Switch
          aria-label={t("games.editor.embed.toggle")}
          checked={panel.on}
          onChange={(e) => panel.setOn(e.currentTarget.checked)}
          disabled={panel.loading || panel.saving || (!panel.on && panel.blocked)}
        />
      }
    >
      {panel.noOrganisation && <HelperText c="orange.8">{t("games.editor.embed.noOrganisation")}</HelperText>}
      {panel.noPages && (
        <Stack gap={4}>
          <HelperText c="orange.8">{t("games.editor.embed.noPages")}</HelperText>
          <Anchor
            size="sm"
            renderRoot={(props) => <Link {...props} to="/my-organization/" search={{ tab: "pages" }} />}
          >
            {t("games.editor.embed.registerPage")}
          </Anchor>
        </Stack>
      )}
      {panel.waitingForApproval && (
        <HelperText c="orange.8">{t("games.editor.embed.waitingForApproval")}</HelperText>
      )}
      {panel.noKeys && !panel.noOrganisation && (
        <HelperText c="orange.8">{t("games.editor.embed.noKeys")}</HelperText>
      )}

      {panel.on && (
        <Stack gap="md" mt="xs">
          {panel.pageUrl && (
            <Stack gap={2}>
              <Label>{t("games.editor.embed.page")}</Label>
              <BodyText size="sm">{panel.pageUrl}</BodyText>
              <HelperText>{t("games.editor.embed.pageHint")}</HelperText>
            </Stack>
          )}

          {!panel.noKeys && (
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
              <Label>{t("games.editor.embed.snippet")}</Label>
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
    </SharingBlock>
  );
}
