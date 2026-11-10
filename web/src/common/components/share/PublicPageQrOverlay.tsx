import { IconQrcode } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { buildShareUrl } from "@/common/lib/url";
import { publicWorkshopPath } from "@/common/lib/publicWorkshop";
import { FullscreenQrOverlay } from "./FullscreenQrOverlay";

interface PublicPageQrOverlayProps {
  opened: boolean;
  onClose: () => void;
  slug: string;
  workshopName: string;
  /** Named at the bottom, so a printed notice says who put it up. */
  organizationName?: string;
}

/**
 * The link to a workshop's public page /w/<slug>, full screen. Its texts address
 * whoever reads it, so it also works printed out as a notice.
 */
export function PublicPageQrOverlay({
  opened,
  onClose,
  slug,
  workshopName,
  organizationName,
}: PublicPageQrOverlayProps) {
  const { t } = useTranslation("common");

  return (
    <FullscreenQrOverlay
      opened={opened}
      onClose={onClose}
      title={t("myOrganization.publicPage.shareTitle", { name: workshopName })}
      subtitle={t("myOrganization.publicPage.shareSubtitle", {
        name: workshopName,
      })}
      description={t("myOrganization.publicPage.shareDescription")}
      footer={
        organizationName
          ? t("myOrganization.publicPage.shareOrganization", {
              name: organizationName,
            })
          : undefined
      }
      icon={<IconQrcode size={28} />}
      color="green"
      url={buildShareUrl(publicWorkshopPath(slug))}
      visitable
    />
  );
}
