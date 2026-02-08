import { useTranslation } from "react-i18next";
import type { SceneMessage } from "../types";
import { SceneCard } from "./SceneCard";
import { PlayerAction } from "./PlayerAction";
import { SystemMessage } from "./SystemMessage";
import { SceneDivider } from "./SceneDivider";
import { TypingIndicator } from "./TypingIndicator";
import { PlayerInput } from "./PlayerInput";
import classes from "./GamePlayer.module.css";

interface MessageListProps {
  messages: SceneMessage[];
  isWaitingForResponse: boolean;
  isImageGenerationDisabled: boolean;
  onSendAction: (message: string) => Promise<void>;
  onRetryLastAction: () => void;
}

export function MessageList({
  messages,
  isWaitingForResponse,
  isImageGenerationDisabled,
  onSendAction,
  onRetryLastAction,
}: MessageListProps) {
  const { t } = useTranslation("common");
  const showImages = !isImageGenerationDisabled;
  const elements: React.ReactNode[] = [];

  // Track previous game message's status fields for showing changes
  let previousGameStatusFields: SceneMessage["statusFields"] = undefined;

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
      elements.push(<SystemMessage key={message.id} message={message} />);
    } else {
      if (index > 0 && messages[index - 1]?.type !== "system") {
        elements.push(<SceneDivider key={`divider-${message.id}`} />);
      }
      elements.push(
        <SceneCard
          key={message.id}
          message={message}
          showImages={showImages}
          previousStatusFields={previousGameStatusFields}
        />,
      );
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

  // Show input inline when user can type
  if (!isWaitingForResponse && messages.length > 0) {
    elements.push(
      <div key="inline-input" className={classes.inlineInput}>
        <PlayerInput
          onSend={onSendAction}
          disabled={isWaitingForResponse}
          placeholder={t("gamePlayer.input.placeholder")}
        />
      </div>,
    );
  }

  return <>{elements}</>;
}
