import type { ReactNode } from "react";
import { IconBook2, IconCards, IconPalette, IconShare, IconVolume } from "@tabler/icons-react";
import type { EditorPage } from "../../lib/editorPages";

/** Each section's icon, on the overview and atop its page. */
export const PAGE_ICON: Record<EditorPage, ReactNode> = {
  card: <IconCards size={18} />,
  story: <IconBook2 size={18} />,
  look: <IconPalette size={18} />,
  sound: <IconVolume size={18} />,
  sharing: <IconShare size={18} />,
};
