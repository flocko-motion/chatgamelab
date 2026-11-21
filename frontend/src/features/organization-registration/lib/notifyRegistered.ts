import { notifications } from "@mantine/notifications";
import type { TFunction } from "i18next";

/** The notice after registering an organisation: it exists right away. */
export function notifyOrganisationRegistered(t: TFunction): void {
  notifications.show({
    color: "green",
    title: t("organizationRegistration.registered.title"),
    message: t("organizationRegistration.registered.text"),
    autoClose: 8000,
  });
}
