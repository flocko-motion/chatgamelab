import { Badge, Card, Group, Stack } from "@mantine/core";
import { useTranslation } from "react-i18next";
import type { VerifiedPageListSort } from "@/api/hooks";
import type { VerifiedPageView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import { ActionButton, TextButton } from "@components/buttons";
import { BodyText, HelperText, ItemTitle } from "@components/typography";
import { VERIFIED_PAGE_LIST_PRESET, type VerifiedPageColumnKey } from "../lib/verifiedPageListPresets";
import { useVerifiedPagesApproval } from "../hooks/useVerifiedPagesApproval";
import classes from "./VerifiedPagesApproval.module.css";

type PageColumn = ListColumn<VerifiedPageView, VerifiedPageListSort>;

/**
 * The allow list of pages that embed games: each organisation's verified
 * page, waiting first, with approving it or withdrawing the approval.
 */
export function VerifiedPagesApproval() {
  const { t } = useTranslation("common");
  const { list, pending, setApproved } = useVerifiedPagesApproval();

  const status = (page: VerifiedPageView) => (
    <Badge size="xs" variant="light" color={page.approvedAt ? "green" : "orange"}>
      {page.approvedAt ? t("admin.verifiedPages.approved") : t("admin.verifiedPages.waiting")}
    </Badge>
  );
  const address = (page: VerifiedPageView) => (
    <Stack gap={0} className={classes.page}>
      <BodyText size="sm">{page.pageUrl}</BodyText>
      <HelperText>{t("admin.verifiedPages.callback", { url: page.callbackUrl })}</HelperText>
    </Stack>
  );
  const action = (page: VerifiedPageView) => {
    if (!page.mayApprove) return null;
    return page.approvedAt ? (
      <TextButton onClick={() => setApproved(page.id, false)} disabled={pending === page.id}>
        {t("admin.verifiedPages.withdraw")}
      </TextButton>
    ) : (
      <ActionButton onClick={() => setApproved(page.id, true)} loading={pending === page.id}>
        {t("admin.verifiedPages.approve")}
      </ActionButton>
    );
  };

  const columns: Record<VerifiedPageColumnKey, PageColumn> = {
    organisation: {
      key: "organisation",
      header: t("admin.verifiedPages.organisation"),
      sortBy: "organisation",
      render: (page) => (
        <ItemTitle lines={1}>{page.organisationName}</ItemTitle>
      ),
    },
    page: { key: "page", header: t("admin.verifiedPages.page"), render: address },
    status: { key: "status", header: t("admin.verifiedPages.status"), sortBy: "status", render: status },
    actions: { key: "actions", header: "", interactive: true, render: action },
  };

  const renderCard = (page: VerifiedPageView) => (
    <Card withBorder padding="sm" radius="md">
      <Stack gap="xs">
        <Group justify="space-between" wrap="nowrap" align="flex-start">
          <ItemTitle lines={1}>{page.organisationName}</ItemTitle>
          {status(page)}
        </Group>
        {address(page)}
        {action(page)}
      </Stack>
    </Card>
  );

  return (
    <ListTable
      compact
      list={list}
      columns={VERIFIED_PAGE_LIST_PRESET.columns.map((key) => columns[key])}
      getRowKey={(page) => page.id}
      renderCard={renderCard}
      empty={{ title: t("admin.verifiedPages.empty") }}
      errorMessage={t("admin.verifiedPages.loadError")}
    />
  );
}
