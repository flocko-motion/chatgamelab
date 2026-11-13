import { Alert, Button, Group, Stack, Text, Textarea, Title } from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { RoutesInstitutionDeclarationResponse } from "@/api/generated";

/** Age label of a suggestion, written so organisations working with children find theirs. */
function suggestionLabelKey(minAge: number): string {
  if (minAge >= 18) return "organizationRegistration.youthProtection.suggestionAdults";
  if (minAge >= 14) return "organizationRegistration.youthProtection.suggestionTeens";
  return "organizationRegistration.youthProtection.suggestionChildren";
}

interface YouthProtectionFieldProps {
  /** Declaration endpoint response: suggestions, 13+ default and length limit. */
  legal: RoutesInstitutionDeclarationResponse;
  value: string;
  onChange: (value: string) => void;
  /** Show "required" when empty (after a first submit attempt). */
  showErrors: boolean;
  /** Smaller heading, e.g. inside a modal. */
  compact?: boolean;
  /** Optional text above the suggestions (replaces the default intro). */
  intro?: string;
}

/**
 * The youth-protection prompt restriction of an organisation: suggestion buttons
 * (13+ default and the seven age levels), the editable text with counter, and both notes.
 * Used at registration and when an organisation head confirms the declaration.
 */
export function YouthProtectionField({
  legal,
  value,
  onChange,
  showErrors,
  compact = false,
  intro,
}: YouthProtectionFieldProps) {
  const { t } = useTranslation("auth");
  const maxLength = legal.maxPromptConstraintLength ?? 200;
  const length = value.trim().length;
  const tooLong = length > maxLength;
  const missing = showErrors && length === 0;

  return (
    <Stack gap="sm">
      <Title order={compact ? 5 : 4}>{t("organizationRegistration.youthProtection.title")}</Title>
      <Text size="sm">{intro ?? t("organizationRegistration.youthProtection.intro")}</Text>
      <Text size="sm" fw={500}>
        {t("organizationRegistration.youthProtection.suggestionsLabel")}
      </Text>
      <Group gap="xs">
        {legal.defaultPromptConstraint && (
          <Button
            size="xs"
            variant={value === legal.defaultPromptConstraint ? "filled" : "outline"}
            onClick={() => onChange(legal.defaultPromptConstraint ?? "")}
          >
            {t("organizationRegistration.youthProtection.defaultOption")}
          </Button>
        )}
        {(legal.youthProtectionSuggestions ?? []).map((s) => (
          <Button
            key={s.minAge}
            size="xs"
            variant={value === s.text ? "filled" : "outline"}
            onClick={() => onChange(s.text ?? "")}
          >
            {t(suggestionLabelKey(s.minAge ?? 0), { age: s.minAge })}
          </Button>
        ))}
      </Group>
      <Textarea
        label={t("organizationRegistration.youthProtection.fieldLabel")}
        value={value}
        onChange={(e) => onChange(e.currentTarget.value)}
        autosize
        minRows={3}
        required
        error={
          missing
            ? t("organizationRegistration.errors.required")
            : tooLong
              ? t("organizationRegistration.youthProtection.tooLong", { max: maxLength })
              : undefined
        }
        description={`${length} / ${maxLength}`}
      />
      <Alert color="blue" variant="light" icon={<IconInfoCircle size={16} />}>
        <Stack gap={4}>
          <Text size="sm">{t("organizationRegistration.youthProtection.changeHint")}</Text>
          <Text size="sm">{t("organizationRegistration.youthProtection.responsibilityHint")}</Text>
        </Stack>
      </Alert>
    </Stack>
  );
}
