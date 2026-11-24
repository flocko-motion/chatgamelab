import { Box, Menu } from "@mantine/core";
import { useBranding } from "@/providers/branding";
import { IconChevronDown, IconFlag, IconPlus, IconTools, IconUpload } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";

export interface NewGameMenuProps {
  onCreate: () => void;
  onImport: () => void;
}

/**
 * The gallery's one action: New game ▾, to create a game, import one, or
 * report a game. On a phone the tab bar's Create takes its place.
 */
export function NewGameMenu({ onCreate, onImport }: NewGameMenuProps) {
  const { t } = useTranslation("common");
  const branding = useBranding();
  return (
    <Box visibleFrom="sm">
      <Menu position="bottom-end" withinPortal>
        <Menu.Target>
          <div>
            <ActionButton leftSection={<IconPlus size={16} />} rightSection={<IconChevronDown size={14} />}>
              {t("games.newGame")}
            </ActionButton>
          </div>
        </Menu.Target>
        <Menu.Dropdown>
          <Menu.Item leftSection={<IconTools size={16} />} onClick={onCreate}>
            {t("games.createButton")}
          </Menu.Item>
          <Menu.Item leftSection={<IconUpload size={16} />} onClick={onImport}>
            {t("games.importExport.importButton")}
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item
            leftSection={<IconFlag size={16} />}
            component="a"
            href={branding.reportGameUrl}
            target="_blank"
            rel="noopener noreferrer"
          >
            {t("allGames.reportGame")}
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>
    </Box>
  );
}
