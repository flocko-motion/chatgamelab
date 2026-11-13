import { notifications } from "@mantine/notifications";
import type { TFunction } from "i18next";

/** Success notice after a registration or upgrade: the organisation exists right away. */
export function notifyOrganisationRegistered(t: TFunction): void {
  notifications.show({
    color: "green",
    title: t("organizationRegistration.registered.title"),
    message: t("organizationRegistration.registered.text"),
    autoClose: 8000,
  });
}
