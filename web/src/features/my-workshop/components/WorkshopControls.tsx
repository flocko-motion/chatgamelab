import { Group } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";
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
  hasGames: boolean;
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
  hasGames,
}: WorkshopControlsProps) {
  const { t } = useTranslation("common");
  const { t: tWorkshop } = useTranslation("myWorkshop");

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

  return (
    <Group justify="space-between" wrap="wrap" gap="sm">
      <Group gap="sm">
        <ActionButton onClick={onCreateClick}>
          {t("games.createButton")}
        </ActionButton>
        <ActionButton onClick={onImportClick}>
          {t("games.importExport.importButton")}
        </ActionButton>
        <ExpandableSearch
          value={searchQuery}
          onChange={onSearchChange}
          placeholder={t("search")}
        />
      </Group>
      <Group gap="sm" wrap="wrap">
        <FilterSegmentedControl
          value={gameFilter}
          onChange={(val) => onFilterChange(val as GameFilter)}
          options={filterOptions}
        />
        {hasGames && (
          <SortSelector
            options={sortOptions}
            value={sortValue}
            onChange={onSortChange}
            label={t("games.sort.label")}
          />
        )}
      </Group>
    </Group>
  );
}
