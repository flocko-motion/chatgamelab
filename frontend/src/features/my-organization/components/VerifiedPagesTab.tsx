import {
  Badge,
  Card,
  Code,
  CopyButton,
  Group,
  Modal,
  Stack,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { IconCheck, IconCopy, IconWorld } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import {
  ActionButton,
  CancelButton,
  DangerButton,
  DeleteIconButton,
  EditIconButton,
  GenericIconButton,
} from "@components/buttons";
import { BodyText, HelperText, ItemTitle, Label } from "@components/typography";
import type { VerifiedPageView } from "@/api/types";
import type { VerifiedPagesTabState } from "../hooks/useVerifiedPagesTab";
import classes from "./VerifiedPagesTab.module.css";

/**
 * The organisation's verified pages: the pages on its website that may
 * frame its embedded games, each asked through its callback before a game
 * starts there. Each shows its page ID, which the callback receives.
 */
export function VerifiedPagesTab({ state }: { state: VerifiedPagesTabState }) {
  const { t } = useTranslation("common");
  const { draft } = state;

  return (
    <Stack gap="md">
      <HelperText>{t("myOrganization.verifiedPages.intro")}</HelperText>

      {state.pages.length === 0 ? (
        <HelperText>{t("myOrganization.verifiedPages.empty")}</HelperText>
      ) : (
        state.pages.map((page) => (
          <PageCard
            key={page.id}
            page={page}
            onEdit={() => state.startEdit(page)}
            onRemove={() => state.askRemove(page)}
          />
        ))
      )}

      {(state.mayRegister || state.editing) && (
        <Card withBorder radius="md" padding="md">
          <Stack gap="sm">
            <Label>
              {state.editing
                ? t("myOrganization.verifiedPages.editTitle")
                : t("myOrganization.verifiedPages.addTitle")}
            </Label>
            {state.editing?.approvedAt && (
              <HelperText c="orange.8">{t("myOrganization.verifiedPages.editReapproval")}</HelperText>
            )}
            <TextInput
              label={t("myOrganization.verifiedPages.pageUrl")}
              description={t("myOrganization.verifiedPages.pageUrlHint")}
              placeholder="https://example.org/play"
              value={draft.pageUrl}
              onChange={(e) => draft.setPageUrl(e.currentTarget.value)}
            />
            <TextInput
              label={t("myOrganization.verifiedPages.callbackUrl")}
              description={t("myOrganization.verifiedPages.callbackUrlHint")}
              placeholder="https://example.org/chatgamelab-callback"
              value={draft.callbackUrl}
              onChange={(e) => draft.setCallbackUrl(e.currentTarget.value)}
            />
            <Group justify="flex-end">
              {state.editing && (
                <CancelButton onClick={state.cancelEdit}>{t("cancel")}</CancelButton>
              )}
              <ActionButton
                onClick={() => void state.submit()}
                disabled={!draft.valid}
                loading={state.submitting}
              >
                {state.editing
                  ? t("myOrganization.verifiedPages.save")
                  : t("myOrganization.verifiedPages.add")}
              </ActionButton>
            </Group>
          </Stack>
        </Card>
      )}

      <Modal
        opened={state.removing !== null}
        onClose={state.cancelRemove}
        title={t("myOrganization.verifiedPages.removeTitle")}
      >
        <Stack gap="md">
          <BodyText size="sm">
            {t("myOrganization.verifiedPages.removeWarning", {
              page: state.removing?.pageUrl ?? "",
            })}
          </BodyText>
          <Group justify="flex-end">
            <CancelButton onClick={state.cancelRemove}>
              {t("cancel")}
            </CancelButton>
            <DangerButton
              onClick={() => void state.confirmRemove()}
              loading={state.removePending}
            >
              {t("myOrganization.verifiedPages.remove")}
            </DangerButton>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}

function PageCard({
  page,
  onEdit,
  onRemove,
}: {
  page: VerifiedPageView;
  onEdit: () => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation("common");
  return (
    <Card withBorder radius="md" padding="md">
      <Group justify="space-between" align="flex-start" wrap="nowrap">
        <Stack gap={4} style={{ minWidth: 0, flex: 1 }}>
          <Group gap="xs" wrap="nowrap" style={{ minWidth: 0 }}>
            <IconWorld size={16} style={{ flexShrink: 0 }} />
            <div className={classes.address}>
              <ItemTitle>{page.pageUrl}</ItemTitle>
            </div>
            <Badge
              size="xs"
              variant="light"
              color={page.approvedAt ? "green" : "orange"}
            >
              {page.approvedAt
                ? t("myOrganization.verifiedPages.approved")
                : t("myOrganization.verifiedPages.waiting")}
            </Badge>
          </Group>
          <Group gap="xs" wrap="nowrap">
            <HelperText>{t("myOrganization.verifiedPages.pageId")}</HelperText>
            <Code>{page.pageId}</Code>
            <CopyButton value={page.pageId}>
              {({ copied, copy }) => (
                <Tooltip
                  label={
                    copied
                      ? t("myOrganization.verifiedPages.copied")
                      : t("myOrganization.verifiedPages.copy")
                  }
                >
                  <GenericIconButton
                    size="sm"
                    onClick={copy}
                    aria-label={t("myOrganization.verifiedPages.copy")}
                    icon={
                      copied ? <IconCheck size={14} /> : <IconCopy size={14} />
                    }
                  />
                </Tooltip>
              )}
            </CopyButton>
          </Group>
          <div className={classes.address}>
            <HelperText>
              {t("myOrganization.verifiedPages.callbackShown", {
                url: page.callbackUrl,
              })}
            </HelperText>
          </div>
        </Stack>
        <Group gap={4} wrap="nowrap">
          {page.mayEdit && (
            <EditIconButton
              onClick={onEdit}
              aria-label={t("myOrganization.verifiedPages.edit")}
            />
          )}
          {page.mayDelete && (
            <DeleteIconButton
              onClick={onRemove}
              aria-label={t("myOrganization.verifiedPages.remove")}
            />
          )}
        </Group>
      </Group>
    </Card>
  );
}
