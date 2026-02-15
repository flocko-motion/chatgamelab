import { Alert } from "@mantine/core";
import { IconKeyOff } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { SceneMessage, PlayerActionInput } from "../types";
import { SceneCard } from "./SceneCard";
import { PlayerAction } from "./PlayerAction";
import { SceneDivider } from "./SceneDivider";
import { TypingIndicator } from "./TypingIndicator";
import { PlayerInput } from "./PlayerInput";
import classes from "./GamePlayer.module.css";

interface MessageListProps {
  messages: SceneMessage[];
  isWaitingForResponse: boolean;
  isImageGenerationDisabled: boolean;
  isAudioMuted: boolean;
  apiKeyUnavailable?: boolean;
  audioEnabled?: boolean;
  onSendAction: (input: PlayerActionInput) => Promise<void>;
  onRetryLastAction: () => void;
}

export function MessageList({
  messages,
  isWaitingForResponse,
  isImageGenerationDisabled,
  isAudioMuted,
  apiKeyUnavailable,
  audioEnabled,
  onSendAction,
  onRetryLastAction,
}: MessageListProps) {
  const { t } = useTranslation("common");
  const showImages = !isImageGenerationDisabled;
  const elements: React.ReactNode[] = [];

  // Track previous game message's status fields for showing changes
  let previousGameStatusFields: SceneMessage["statusFields"] = undefined;

  // Collect system prompt text from system messages for the first game message
  let systemPromptText = "";
  let isFirstGameMessage = true;

  messages.forEach((message, index) => {
    if (message.type === "player") {
      elements.push(
        <PlayerAction
          key={message.id}
          text={message.text}
          error={message.error}
          errorCode={message.errorCode}
          onRetry={message.error ? onRetryLastAction : undefined}
        />,
      );
    } else if (message.type === "system") {
      systemPromptText += (systemPromptText ? "\n\n" : "") + message.text;
    } else {
      if (index > 0 && messages[index - 1]?.type !== "system") {
        elements.push(<SceneDivider key={`divider-${message.id}`} />);
      }
      elements.push(
        <SceneCard
          key={message.id}
          message={message}
          showImages={showImages}
          isAudioMuted={isAudioMuted}
          previousStatusFields={previousGameStatusFields}
          systemPrompt={isFirstGameMessage ? systemPromptText : undefined}
          isFirstGameMessage={isFirstGameMessage}
        />,
      );
      isFirstGameMessage = false;
      // Update previous status fields for next game message
      if (message.statusFields?.length) {
        previousGameStatusFields = message.statusFields;
      }
    }
  });

  if (isWaitingForResponse && messages.length > 0) {
    const lastMessage = messages[messages.length - 1];
    if (lastMessage.type === "player" || !lastMessage.isStreaming) {
      elements.push(<TypingIndicator key="typing" />);
    }
  }

  // Show input inline when user can type, or a banner if API key is unavailable
  if (!isWaitingForResponse && messages.length > 0) {
    if (apiKeyUnavailable) {
      elements.push(
        <div key="api-key-banner" className={classes.inlineInput}>
          <Alert
            variant="filled"
            color="red"
            icon={<IconKeyOff size={18} />}
            radius="md"
          >
            {t("gamePlayer.error.noApiKey.banner")}
          </Alert>
        </div>,
      );
    } else {
      elements.push(
        <div key="inline-input" className={classes.inlineInput}>
          <PlayerInput
            onSend={onSendAction}
            disabled={isWaitingForResponse}
            placeholder={t("gamePlayer.input.placeholder")}
            audioEnabled={audioEnabled}
          />
        </div>,
      );
    }
  }

  return <>{elements}</>;
}
