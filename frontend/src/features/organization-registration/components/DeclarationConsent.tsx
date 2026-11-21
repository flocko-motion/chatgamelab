import { Alert, Checkbox, Stack } from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { HelperText } from "@components/typography";
import type { DeclarationView } from "@/api/generated/api.gen";
import { DeclarationText } from "./DeclarationText";

interface DeclarationConsentProps {
  declaration: DeclarationView;
  accepted: boolean;
  onChange: (accepted: boolean) => void;
  error?: string;
}

/**
 * The declaration with its version, and the box that agrees to it. Both are
 * binding in German: the text and the sentence ticked come from the server
 * and stay German in every interface language, with a note where it is not.
 */
export function DeclarationConsent({ declaration, accepted, onChange, error }: DeclarationConsentProps) {
  const { t, i18n } = useTranslation("auth");
  const isGermanUi = (i18n.resolvedLanguage ?? i18n.language ?? "").startsWith("de");
  return (
    <Stack gap="sm">
      {!isGermanUi && (
        <Alert color="gray" variant="light" icon={<IconInfoCircle size={16} />}>
          {t("organizationRegistration.declarationGermanOnly")}
        </Alert>
      )}
      <DeclarationText text={declaration.text} />
      <HelperText>
        {t("organizationRegistration.declarationVersion", {
          version: declaration.version,
          date: new Date(declaration.date).toLocaleDateString(i18n.language),
        })}
      </HelperText>
      <Checkbox
        lang="de"
        label={declaration.consent}
        checked={accepted}
        onChange={(e) => onChange(e.currentTarget.checked)}
        error={error}
      />
    </Stack>
  );
}
