import { Alert, Chip, Group, Stack, Textarea } from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { BodyText, CardTitle } from "@components/typography";
import type { DeclarationResp } from "@/api/generated/api.gen";

/** The label of a suggestion, worded so organisations of children find theirs. */
function suggestionLabelKey(minAge: number): string {
  if (minAge >= 18) return "organizationRegistration.youthProtection.suggestionAdults";
  if (minAge >= 14) return "organizationRegistration.youthProtection.suggestionTeens";
  return "organizationRegistration.youthProtection.suggestionChildren";
}

interface YouthProtectionFieldProps {
  /** The declaration's answer: the suggestions, the 13+ default and the longest restriction. */
  declaration: DeclarationResp;
  value: string;
  onChange: (value: string) => void;
  /** Show "required" when empty, after a first attempt to send. */
  showErrors: boolean;
  /** Text above the suggestions instead of the general one. */
  intro?: string;
}

/**
 * An organisation's youth-protection restriction: the suggestions (the 13+
 * default and the seven age levels), the text with its counter, and the
 * notes on changing it and on who is responsible. Shown when an organisation
 * registers and when a head consents to the declaration.
 */
export function YouthProtectionField({ declaration, value, onChange, showErrors, intro }: YouthProtectionFieldProps) {
  const { t } = useTranslation("auth");
  const max = declaration.maxPromptConstraintLength;
  const length = value.trim().length;
  const suggestions = [
    ...(declaration.defaultPromptConstraint
      ? [{ text: declaration.defaultPromptConstraint, label: t("organizationRegistration.youthProtection.defaultOption") }]
      : []),
    ...declaration.youthProtectionSuggestions.map((s) => ({
      text: s.text,
      label: t(suggestionLabelKey(s.minAge), { age: s.minAge }),
    })),
  ];

  let error: string | undefined;
  if (showErrors && length === 0) error = t("organizationRegistration.errors.required");
  else if (length > max) error = t("organizationRegistration.youthProtection.tooLong", { max });

  return (
    <Stack gap="sm">
      <CardTitle>{t("organizationRegistration.youthProtection.title")}</CardTitle>
      <BodyText size="sm">{intro ?? t("organizationRegistration.youthProtection.intro")}</BodyText>
      <BodyText size="sm" fw={500}>
        {t("organizationRegistration.youthProtection.suggestionsLabel")}
      </BodyText>
      <Group gap="xs">
        {suggestions.map((s) => (
          <Chip key={s.label} size="xs" checked={value === s.text} onChange={() => onChange(s.text)}>
            {s.label}
          </Chip>
        ))}
      </Group>
      <Textarea
        label={t("organizationRegistration.youthProtection.fieldLabel")}
        value={value}
        onChange={(e) => onChange(e.currentTarget.value)}
        autosize
        minRows={3}
        required
        error={error}
        description={`${length} / ${max}`}
      />
      <Alert color="blue" variant="light" icon={<IconInfoCircle size={16} />}>
        <Stack gap={4}>
          <BodyText size="sm">{t("organizationRegistration.youthProtection.changeHint")}</BodyText>
          <BodyText size="sm">{t("organizationRegistration.youthProtection.responsibilityHint")}</BodyText>
        </Stack>
      </Alert>
    </Stack>
  );
}
