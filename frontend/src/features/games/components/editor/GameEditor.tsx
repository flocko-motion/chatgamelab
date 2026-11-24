import { useEffect, useRef, type ReactNode } from "react";
import { useBranding } from "@/providers/branding";
import { ActionIcon, Alert, Badge, Group, Modal, Skeleton, Stack, Text, ThemeIcon } from "@mantine/core";
import { IconAlertCircle, IconAlertTriangle, IconArrowLeft, IconCopy, IconHeartFilled, IconSparkles } from "@tabler/icons-react";
import { useModals } from "@mantine/modals";
import { useTranslation } from "react-i18next";
import type { CreateGameReq, GameView } from "@/api/types";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { ActionButton, CancelButton, GenericButtonWithText } from "@components/buttons";
import { HelpLink } from "@components/HelpLink";
import { HelperText, Label, SectionTitle } from "@components/typography";
import type { CreateGameFormData } from "../../types";
import { useGameEditor, type GameEditor as Editor } from "../../hooks/useGameEditor";
import { useEditorNavigation, type EditorNavigation, type EditorView } from "../../hooks/useEditorNavigation";
import { useVoicePreview, type VoicePreview } from "../../hooks/useVoicePreview";
import { useGameReview } from "../../hooks/useGameReview";
import { hasCustomVoice, useVoiceSlots } from "../../hooks/useGenreFields";
import { CardSection, LookSection, StorySection } from "./EditorSections";
import { SoundSection } from "./SoundSection";
import { SharingSection } from "./SharingSection";
import { DeletePage } from "./DeletePage";
import { EditorOverview } from "./EditorOverview";
import { GenreQuestion } from "./GenreQuestion";
import { ReviewDialog } from "./ReviewDialog";
import { PAGE_ICON } from "./pageIcons";
import classes from "./GameEditor.module.css";

export interface GameEditorProps {
  /** The game to open; none makes a new one. */
  gameId?: string | null;
  opened: boolean;
  onClose: () => void;
  /**
   * Creates the game: the template values with the form changes. Answers
   * with the game as created, so the editor can have its cover drawn.
   */
  onCreate?: (request: CreateGameReq) => Promise<GameView | undefined> | void;
  createLoading?: boolean;
  /** Pre-populates a new game: an import, or a copy. */
  initialData?: Partial<CreateGameFormData> | null;
  /** The new game is the caller's copy of someone else's game. */
  saveAsOwn?: boolean;
  /** Makes the caller's own copy of a game they may only look at. */
  onCopy?: () => void;
  onSponsor?: () => void;
  /** Whether the list the editor opened from offers share links. */
  showShareSection?: boolean;
  workshopId?: string;
  /** The caller made the game (a fact, for the admin warning). */
  isOwner?: boolean;
  /** Someone else's game of the caller's workshop (a fact, for the delete page). */
  isOthersWorkshopGame?: boolean;
}

/**
 * The game editor: an overview of sections, one page each, opened in any
 * order. A new game starts with its genre; someone else's game opens as
 * itself, to look at, with a copy of one's own a button away. One fixed
 * width, centred; on a phone, the whole screen. Only the middle scrolls.
 */
export function GameEditor(props: GameEditorProps) {
  const { isMobile } = useResponsiveDesign();
  // Closing asks first when something is unsaved; the window inside knows.
  const requestClose = useRef<() => void>(props.onClose);
  return (
    <Modal.Root
      opened={props.opened}
      onClose={() => requestClose.current()}
      size={980}
      fullScreen={isMobile}
      centered
      classNames={{ content: `${classes.window} ${isMobile ? "" : classes.windowFixed}` }}
    >
      <Modal.Overlay />
      {/* Mounted while open, so every opening starts afresh. */}
      {props.opened && (
        <EditorWindow
          {...props}
          phone={isMobile}
          onCloseRequest={(close) => {
            requestClose.current = close;
          }}
        />
      )}
    </Modal.Root>
  );
}

function EditorWindow(
  props: GameEditorProps & { phone: boolean; onCloseRequest: (close: () => void) => void },
) {
  const { t } = useTranslation("common");
  const branding = useBranding();
  const { phone } = props;
  const modals = useModals();
  const editor = useGameEditor({
    gameId: props.gameId,
    opened: props.opened,
    initialData: props.initialData,
    onCreate: props.onCreate,
    onDone: props.onClose,
    isOwner: props.isOwner,
  });
  const nav = useEditorNavigation(editor);
  const voiceSlots = useVoiceSlots(editor.values.genre);
  const preview = useVoicePreview(
    editor.values.genre,
    editor.values.voiceStyle,
    editor.values.systemMessageScenario,
    hasCustomVoice(editor.values.voices, voiceSlots),
  );
  const review = useGameReview(editor.values);
  const { adminAction } = editor;

  // Every page opens at its top.
  const body = useRef<HTMLDivElement>(null);
  const viewKey = nav.view.kind === "page" ? nav.view.page : nav.view.kind;
  useEffect(() => {
    body.current?.scrollTo({ top: 0 });
  }, [viewKey]);

  const close = () => {
    preview.stop();
    if (!editor.dirty) return props.onClose();
    modals.openConfirmModal({
      title: t("games.editModal.unsavedChanges.title"),
      children: <Text size="sm">{t("games.editModal.unsavedChanges.message")}</Text>,
      labels: {
        confirm: t("games.editModal.unsavedChanges.discard"),
        cancel: t("games.editModal.unsavedChanges.keepEditing"),
      },
      confirmProps: { color: "red" },
      onConfirm: props.onClose,
    });
  };

  const { onCloseRequest } = props;
  useEffect(() => {
    onCloseRequest(close);
  });

  let title = t("games.editor.titleEdit");
  if (editor.creating && !props.saveAsOwn) title = t("games.editor.titleNew");
  else if (editor.readOnly) title = t("games.editor.titleView");

  return (
    <Modal.Content className={adminAction ? classes.adminAction : undefined}>
      <Modal.Header>
        <Modal.Title>
          <Group gap="sm">
            <Label>{title}</Label>
            <HelpLink href={branding.gameTipsUrl} />
            {editor.game?.publicSponsoredApiKeyShareId && (
              <Badge size="sm" color="pink" variant="light" leftSection={<IconHeartFilled size={12} />}>
                {t("games.sponsor.sponsored")}
              </Badge>
            )}
          </Group>
        </Modal.Title>
        <Modal.CloseButton aria-label={t("close")} />
      </Modal.Header>
      <Modal.Body ref={body} className={classes.body}>
        <Body {...props} editor={editor} nav={nav} preview={preview} adminAction={adminAction} />
      </Modal.Body>
      {nav.view.kind !== "genre" && !editor.loading && !editor.failed && (
        <div className={classes.footer}>
          <Footer
            view={nav.view}
            editor={editor}
            nav={nav}
            phone={phone}
            saveAsOwn={!!props.saveAsOwn}
            saving={editor.creating ? !!props.createLoading : editor.saving}
            reviewing={review.asking}
            onReview={review.ask}
            onCancel={close}
            onCopy={editor.game?.mayCopy ? props.onCopy : undefined}
          />
        </div>
      )}
      <ReviewDialog
        note={review.note}
        onClose={review.dismiss}
        onGo={(page) => {
          review.dismiss();
          nav.open(page);
        }}
      />
    </Modal.Content>
  );
}

interface BodyProps extends GameEditorProps {
  editor: Editor;
  nav: EditorNavigation;
  preview: VoicePreview;
  adminAction: boolean;
}

function Body({ editor, nav, preview, adminAction, ...props }: BodyProps & { phone?: boolean }) {
  const { t } = useTranslation("common");
  const { view } = nav;

  if (view.kind === "genre") {
    return <GenreQuestion loading={editor.loading} onPick={nav.pickGenre} />;
  }
  if (editor.loading) {
    return (
      <Stack gap="md" py="sm">
        <Skeleton height={32} width="60%" />
        <Skeleton height={120} />
        <Skeleton height={120} />
      </Stack>
    );
  }
  if (editor.failed) {
    return (
      <Alert icon={<IconAlertCircle size={16} />} color="red">
        {t("games.errors.loadFailed")}
      </Alert>
    );
  }

  if (view.kind === "delete" && editor.game) {
    return (
      <PageFrame title={t("games.editor.delete.label")} onBack={nav.back} danger>
        <DeletePage
          game={editor.game}
          name={editor.values.name}
          deleting={editor.removing}
          onDelete={editor.remove}
          othersWorkshopGame={!!props.isOthersWorkshopGame}
          adminAction={adminAction}
        />
      </PageFrame>
    );
  }

  if (view.kind === "page") {
    const page = view.page;
    let section: ReactNode = null;
    switch (page) {
      case "card":
        section = <CardSection editor={editor} shows={nav.shows} />;
        break;
      case "story":
        section = <StorySection editor={editor} shows={nav.shows} />;
        break;
      case "look":
        section = <LookSection editor={editor} shows={nav.shows} />;
        break;
      case "sound":
        section = <SoundSection editor={editor} preview={preview} />;
        break;
      case "sharing":
        section = editor.game ? (
          <SharingSection
            editor={editor}
            game={editor.game}
            showShareLinks={!!props.showShareSection}
            workshopId={props.workshopId}
            onSponsor={props.onSponsor}
          />
        ) : null;
        break;
    }
    return (
      <PageFrame
        title={t(`games.editor.pages.${page}.label`)}
        intro={t(`games.editor.pages.${page}.intro`)}
        icon={PAGE_ICON[page]}
        onBack={nav.back}
      >
        {section}
      </PageFrame>
    );
  }

  return (
    <Stack gap="md">
      {adminAction && (
        <Alert color="red" icon={<IconAlertTriangle size={16} />} title={t("games.editModal.adminActionTitle")}>
          {t("games.editModal.adminActionWarning")}
        </Alert>
      )}
      <EditorOverview editor={editor} nav={nav} preview={preview} phone={props.phone ?? false} />
    </Stack>
  );
}

/** A section's page: back to the overview at the top left, its name, what it is for. */
function PageFrame({
  title,
  intro,
  icon,
  danger,
  onBack,
  children,
}: {
  title: string;
  intro?: string;
  icon?: ReactNode;
  danger?: boolean;
  onBack: () => void;
  children: ReactNode;
}) {
  const { t } = useTranslation("common");
  return (
    <Stack gap="lg" py="sm">
      <Group gap="sm" wrap="nowrap">
        <ActionIcon variant="subtle" size="lg" onClick={onBack} aria-label={t("games.editor.backToOverview")}>
          <IconArrowLeft />
        </ActionIcon>
        <SectionTitle accent={!danger}>{title}</SectionTitle>
      </Group>
      {intro && (
        <Group gap="xs" wrap="nowrap">
          <ThemeIcon variant="light" radius="md">
            {icon}
          </ThemeIcon>
          <HelperText>{intro}</HelperText>
        </Group>
      )}
      {children}
    </Stack>
  );
}

interface FooterProps {
  view: EditorView;
  editor: Editor;
  nav: EditorNavigation;
  phone: boolean;
  saveAsOwn: boolean;
  saving: boolean;
  reviewing: boolean;
  onReview: () => void;
  onCancel: () => void;
  onCopy?: () => void;
}

/**
 * The buttons below the scrolling middle. A section: Done. Deleting: Keep
 * it, where Done and Save sit elsewhere. The overview: Cancel, Review and
 * Save (on a phone the × cancels); someone else's game: Close and Make my
 * copy.
 */
function Footer({ view, editor, nav, phone, saveAsOwn, saving, reviewing, onReview, onCancel, onCopy }: FooterProps) {
  const { t } = useTranslation("common");
  let buttons: ReactNode;
  if (view.kind === "page") {
    buttons = <ActionButton onClick={nav.back}>{t("games.editor.doneButton")}</ActionButton>;
  } else if (view.kind === "delete") {
    buttons = <ActionButton onClick={nav.back}>{t("games.editor.delete.keep")}</ActionButton>;
  } else if (editor.readOnly) {
    buttons = (
      <>
        {!phone && <CancelButton onClick={onCancel}>{t("close")}</CancelButton>}
        {onCopy && (
          <ActionButton onClick={onCopy} leftSection={<IconCopy size={16} />}>
            {t("games.editor.makeCopy")}
          </ActionButton>
        )}
      </>
    );
  } else {
    let save = t("save");
    if (saveAsOwn) save = t("games.editModal.saveAsOwn");
    else if (editor.creating) save = t("games.createModal.submit");
    buttons = (
      <>
        {!phone && (
          <CancelButton onClick={onCancel} disabled={saving}>
            {t("cancel")}
          </CancelButton>
        )}
        <GenericButtonWithText
          icon={<IconSparkles size={16} />}
          variant="default"
          size="sm"
          onClick={onReview}
          loading={reviewing}
        >
          {t("games.editor.review.button")}
        </GenericButtonWithText>
        <ActionButton onClick={editor.save} loading={saving} disabled={editor.creating && !nav.allDone}>
          {save}
        </ActionButton>
      </>
    );
  }
  return (
    <Group justify="flex-end" gap="sm" wrap="nowrap">
      {buttons}
    </Group>
  );
}
