import { UnstyledButton } from "@mantine/core";
import { IconSchool } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ItemTitle } from "@components/typography";
import type { DashboardData } from "../hooks/useDashboard";
import { SectionHead, SeeAll } from "./GameSections";
import classes from "./Dashboard.module.css";

/** "Your lab so far": the platform's games, the user's own games and their plays; then their workshops. */
export function LabPanel({ data }: { data: DashboardData }) {
  const { t } = useTranslation("dashboard");
  const { numbers, workshops } = data;
  const show = (n: number | undefined) => (n === undefined ? "–" : n.toLocaleString());
  return (
    <section>
      <SectionHead title={t("lab.title")} />
      <dl className={classes.numbers} style={{ margin: 0 }}>
        <div className={classes.number} data-wide>
          <dt>{t("lab.gamesOnPlatform")}</dt>
          <dd style={{ margin: 0 }}><b>{show(numbers.gamesOnPlatform)}</b></dd>
        </div>
        <div className={classes.number} data-own>
          <dt>{t("lab.gamesCreated")}</dt>
          <dd style={{ margin: 0 }}><b>{show(numbers.gamesCreated)}</b></dd>
        </div>
        <div className={classes.number} data-own>
          <dt>{t("lab.playsOfOwnGames")}</dt>
          <dd style={{ margin: 0 }}><b>{show(numbers.playsOfOwnGames)}</b></dd>
        </div>
      </dl>

      {workshops.length > 0 && (
        <div className={classes.workshopList}>
          <SectionHead
            title={t("workshops.title")}
            action={<SeeAll label={t("workshops.viewAll")} onClick={data.go.workshops} />}
          />
          {workshops.map((workshop) => (
            <UnstyledButton
              key={workshop.id}
              className={classes.workshopRow}
              onClick={() => data.go.workshop(workshop.id ?? "", workshop.name ?? "")}
            >
              <ItemTitle lines={1}>{workshop.name ?? t("untitled")}</ItemTitle>
              <IconSchool size={18} aria-hidden />
            </UnstyledButton>
          ))}
        </div>
      )}
    </section>
  );
}
