import { Button } from "@mantine/core";
import { IconHelp } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

interface HelpLinkProps {
  /** The URL to open */
  href: string;
  /** Optional button text (defaults to "Help") */
  label?: string;
  /** Icon size in px */
  size?: number;
}

/**
 * A compact help button that links to an external help page.
 * Designed to sit in headers, modal titles, or section titles
 * without cluttering the UI.
 *
 * @example
 * <HelpLink href={HELP_LINKS.GAME_TIPS} />
 */
export function HelpLink({ href, label, size = 16 }: HelpLinkProps) {
  const { t } = useTranslation("common");
  const text = label ?? t("help", "Help");

  return (
    <Button
      component="a"
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      variant="filled"
      color="accent"
      size="compact-xs"
      leftSection={<IconHelp size={size} />}
    >
      {text}
    </Button>
  );
}
