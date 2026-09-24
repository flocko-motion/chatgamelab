import { IconQrcode } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { buildShareUrl } from "@/common/lib/url";
import { publicWorkshopPath } from "@/common/lib/publicWorkshop";
import { FullscreenQrOverlay } from "./FullscreenQrOverlay";

interface PublicPageQrOverlayProps {
  opened: boolean;
  onClose: () => void;
  slug: string;
}

/** The link to a workshop's public page /w/<slug>, full screen. */
export function PublicPageQrOverlay({
  opened,
  onClose,
  slug,
}: PublicPageQrOverlayProps) {
  const { t } = useTranslation("common");

  return (
    <FullscreenQrOverlay
      opened={opened}
      onClose={onClose}
      title={t("myOrganization.publicPage.shareTitle")}
      description={t("myOrganization.publicPage.shareDescription")}
      icon={<IconQrcode size={28} />}
      color="green"
      url={buildShareUrl(publicWorkshopPath(slug))}
    />
  );
}
