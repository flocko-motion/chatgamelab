import { useTranslation } from "react-i18next";
import type { ObjWorkshop } from "@/api/generated";
import {
  AiTierLimitNotice,
  AiTierNotice,
} from "@/common/components/AiTierNotice";
import { focusWorkshopKeySelect } from "../lib/workshopKeySelect";

/**
 * Below a workshop's tier select: names the tiers the free-use key it falls
 * back to is not released for, or says that no key at all pays for it.
 */
export function WorkshopTierNotice({ workshop }: { workshop: ObjWorkshop }) {
  const { t } = useTranslation("common");
  const selectKey = () => focusWorkshopKeySelect(workshop.id!);

  if (workshop.apiKeySource === "") {
    return (
      <AiTierNotice
        title={t("aiQualityTier.noKey.title")}
        reason={t("aiQualityTier.noKey.workshop")}
        actionLabel={t("aiQualityTier.notUsed.selectWorkshopKey")}
        onAction={selectKey}
      />
    );
  }
  if (
    workshop.aiQualityTierLimit &&
    (workshop.apiKeySource === "organization" ||
      workshop.apiKeySource === "server")
  ) {
    return (
      <AiTierLimitNotice
        limit={workshop.aiQualityTierLimit}
        source={workshop.apiKeySource}
        audience="workshop"
        actionLabel={t("aiQualityTier.notUsed.selectWorkshopKey")}
        onAction={selectKey}
      />
    );
  }
  return null;
}
