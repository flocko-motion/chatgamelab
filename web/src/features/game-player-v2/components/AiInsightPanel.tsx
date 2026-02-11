import { useState } from "react";
import { Collapse } from "@mantine/core";
import {
  IconEye,
  IconEyeOff,
  IconChevronDown,
  IconChevronRight,
  IconBrain,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { SceneMessage } from "../types";
import classes from "./AiInsightPanel.module.css";

interface AiInsightPanelProps {
  message: SceneMessage;
  /** The system prompt text to show for the first AI message */
  systemPrompt?: string;
  /** Whether this is the first game message (shows system prompt section) */
  isFirstGameMessage?: boolean;
}

interface SectionProps {
  label: string;
  content: string;
  defaultOpen?: boolean;
}

function CollapsibleSection({
  label,
  content,
  defaultOpen = false,
}: SectionProps) {
  const [opened, setOpened] = useState(defaultOpen);

  return (
    <div className={classes.section}>
      <div
        className={classes.sectionLabel}
        onClick={() => setOpened((o) => !o)}
      >
        {opened ? (
          <IconChevronDown size={12} />
        ) : (
          <IconChevronRight size={12} />
        )}
        <span className={classes.sectionLabelText}>{label}</span>
      </div>
      <Collapse in={opened}>
        <div className={classes.sectionContent}>{content}</div>
      </Collapse>
    </div>
  );
}

export function AiInsightPanel({
  message,
  systemPrompt,
  isFirstGameMessage,
}: AiInsightPanelProps) {
  const { t } = useTranslation("common");
  const [opened, setOpened] = useState(false);

  const hasAnyData =
    message.requestExpandStory ||
    message.requestResponseSchema ||
    message.requestStatusUpdate ||
    message.requestImageGeneration ||
    message.responseRaw ||
    (isFirstGameMessage && systemPrompt);

  if (!hasAnyData) return null;

  const usage = message.tokenUsage;
  const hasTokens = usage && (usage.totalTokens ?? 0) > 0;

  return (
    <div className={classes.wrapper}>
      <button
        type="button"
        className={`${classes.toggleButton} ${opened ? classes.toggleButtonActive : ""}`}
        onClick={() => setOpened((o) => !o)}
        title={t("gamePlayer.aiInsight.toggle")}
      >
        {opened ? <IconEyeOff size={15} /> : <IconEye size={15} />}
        {t("gamePlayer.aiInsight.toggle")}
      </button>

      <Collapse in={opened} className={classes.panelCollapse}>
        <div className={classes.panel}>
          <div className={classes.panelHeader}>
            <IconBrain size={14} className={classes.panelHeaderIcon} />
            {t("gamePlayer.aiInsight.title")}
            {hasTokens && (
              <span className={classes.tokenBadge}>
                🔤 {usage.inputTokens?.toLocaleString() ?? 0}{" "}
                {t("gamePlayer.aiInsight.tokens.sent")} ·{" "}
                {usage.outputTokens?.toLocaleString() ?? 0}{" "}
                {t("gamePlayer.aiInsight.tokens.received")}
              </span>
            )}
          </div>
          <div className={classes.sections}>
            {isFirstGameMessage && systemPrompt && (
              <CollapsibleSection
                label={t("gamePlayer.aiInsight.sections.systemPrompt")}
                content={systemPrompt}
                defaultOpen
              />
            )}
            {message.requestResponseSchema && (
              <CollapsibleSection
                label={t("gamePlayer.aiInsight.sections.responseSchema")}
                content={formatJson(message.requestResponseSchema)}
              />
            )}
            {message.requestStatusUpdate && !isFirstGameMessage && (
              <CollapsibleSection
                label={t("gamePlayer.aiInsight.sections.statusUpdate")}
                content={message.requestStatusUpdate}
              />
            )}
            {message.requestExpandStory && (
              <CollapsibleSection
                label={t("gamePlayer.aiInsight.sections.expandStory")}
                content={message.requestExpandStory}
              />
            )}
            {message.responseRaw && (
              <CollapsibleSection
                label={t("gamePlayer.aiInsight.sections.rawResponse")}
                content={formatJson(message.responseRaw)}
              />
            )}
            {message.requestImageGeneration && (
              <CollapsibleSection
                label={t("gamePlayer.aiInsight.sections.imageGeneration")}
                content={message.requestImageGeneration}
              />
            )}
          </div>
        </div>
      </Collapse>
    </div>
  );
}

/** Try to pretty-print JSON strings, fall back to raw string */
function formatJson(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}
