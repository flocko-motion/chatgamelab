import { Group, Tooltip } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { IconFileImport } from "@tabler/icons-react";
import { ActionButton, PlusIconButton, GenericIconButton } from "@components/buttons";
import {
  SortSelector,
  type SortOption,
  FilterSegmentedControl,
  ExpandableSearch,
} from "@components/controls";
import type { GameFilter } from "../types";

interface WorkshopControlsProps {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  gameFilter: GameFilter;
  onFilterChange: (value: GameFilter) => void;
  sortValue: string;
  onSortChange: (value: string) => void;
  onCreateClick: () => void;
  onImportClick: () => void;
}

export function WorkshopControls({
  searchQuery,
  onSearchChange,
  gameFilter,
  onFilterChange,
  sortValue,
  onSortChange,
  onCreateClick,
  onImportClick,
}: WorkshopControlsProps) {
  const { t } = useTranslation("common");
  const { t: tWorkshop } = useTranslation("myWorkshop");
  const isMobile = useMediaQuery("(max-width: 48em)");

  const sortOptions: SortOption[] = [
    { value: "modifiedAt-desc", label: t("games.sort.modifiedAt-desc") },
    { value: "modifiedAt-asc", label: t("games.sort.modifiedAt-asc") },
    { value: "createdAt-desc", label: t("games.sort.createdAt-desc") },
    { value: "createdAt-asc", label: t("games.sort.createdAt-asc") },
    { value: "name-asc", label: t("games.sort.name-asc") },
    { value: "name-desc", label: t("games.sort.name-desc") },
  ];

  const filterOptions = [
    { value: "all", label: tWorkshop("filters.all") },
    { value: "mine", label: tWorkshop("filters.mine") },
    { value: "workshop", label: tWorkshop("filters.workshop") },
    { value: "public", label: tWorkshop("filters.public") },
  ];

  if (isMobile) {
    return (
      <Group gap="sm" wrap="nowrap">
        <Tooltip label={t("games.createButton")} withArrow>
          <PlusIconButton onClick={onCreateClick} variant="filled" aria-label={t("games.createButton")} />
        </Tooltip>
        <Tooltip label={t("games.importExport.importButton")} withArrow>
          <GenericIconButton
            icon={<IconFileImport size={16} />}
            onClick={onImportClick}
            aria-label={t("games.importExport.importButton")}
          />
        </Tooltip>
        <ExpandableSearch
          value={searchQuery}
          onChange={onSearchChange}
          placeholder={t("search")}
        />
        <Group gap="xs" wrap="nowrap" style={{ flexShrink: 0 }}>
          <FilterSegmentedControl
            value={gameFilter}
            onChange={(val) => onFilterChange(val as GameFilter)}
            options={filterOptions}
          />
          <SortSelector
            options={sortOptions}
            value={sortValue}
            onChange={onSortChange}
            label={t("games.sort.label")}
          />
        </Group>
      </Group>
    );
  }

  return (
    <Group justify="space-between" wrap="wrap" gap="sm">
      <Group gap="sm">
        <ActionButton onClick={onCreateClick}>
          {t("games.createButton")}
        </ActionButton>
        <ActionButton onClick={onImportClick}>
          {t("games.importExport.importButton")}
        </ActionButton>
      </Group>
      <Group gap="sm" wrap="wrap">
        <ExpandableSearch
          value={searchQuery}
          onChange={onSearchChange}
          placeholder={t("search")}
        />
        <FilterSegmentedControl
          value={gameFilter}
          onChange={(val) => onFilterChange(val as GameFilter)}
          options={filterOptions}
        />
        <SortSelector
          options={sortOptions}
          value={sortValue}
          onChange={onSortChange}
          label={t("games.sort.label")}
        />
      </Group>
    </Group>
  );
}
