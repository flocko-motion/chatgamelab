import { IconKey } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import classes from "./ApiKeyChangeIndicator.module.css";

export function ApiKeyChangeIndicator() {
  const { t } = useTranslation("common");

  return (
    <div className={classes.container}>
      <span className={classes.line} />
      <span className={classes.label}>
        <IconKey size={12} className={classes.icon} />
        {t("gamePlayer.aiInsight.apiKeyChanged")}
      </span>
      <span className={classes.line} />
    </div>
  );
}
