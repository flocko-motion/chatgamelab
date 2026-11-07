import { useNavigate } from "@tanstack/react-router";
import {
  Alert,
  Card,
  Container,
  Group,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { IconAlertCircle, IconArrowRight } from "@tabler/icons-react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@/common/components/buttons/ActionButton";
import { LanguageSwitcher } from "@/common/components/LanguageSwitcher";
import { config } from "@/config/env";
import { ROUTES } from "@/common/routes/routes";
import { buildShareUrl } from "@/common/lib/url";
import { ErrorCodes } from "@/common/types/errorCodes";
import { storeParticipantToken } from "@/providers/AuthProvider";
import {
  INVITE_WORD_COUNT,
  PARTICIPANT_WORD_COUNT,
  inviteLinkPath,
  normalizeWordToken,
  toParticipantToken,
  wordCount,
} from "@/common/lib/wordToken";

interface EnterCodePageProps {
  /** From a short link /code/<words>: submitted once on arrival. */
  initialCode?: string;
}

/** One field for both codes: 3 words join a workshop, 4 words log back in. */
export function EnterCodePage({ initialCode }: EnterCodePageProps) {
  const { t } = useTranslation("common");
  const navigate = useNavigate();
  const [input, setInput] = useState(initialCode ?? "");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const submittedInitial = useRef(false);

  const submitCode = async (raw: string) => {
    const code = normalizeWordToken(raw);
    const words = wordCount(code);
    setError(null);

    if (words === INVITE_WORD_COUNT) {
      navigate({ to: inviteLinkPath(code), replace: !!initialCode });
      return;
    }
    if (words !== PARTICIPANT_WORD_COUNT) {
      setError(t("enterCode.wrongLength"));
      return;
    }

    setSubmitting(true);
    try {
      const token = toParticipantToken(code);
      const response = await fetch(
        `${config.API_BASE_URL}/auth/participant-login`,
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ token }),
        },
      );
      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        setError(
          data.code === ErrorCodes.TOKEN_LOCKED
            ? t("invites.errors.tokenLocked")
            : t("enterCode.unknown"),
        );
        return;
      }
      storeParticipantToken(token);
      window.location.href = buildShareUrl(ROUTES.MY_WORKSHOP);
    } catch {
      setError(t("invites.errors.loadFailed"));
    } finally {
      setSubmitting(false);
    }
  };

  useEffect(() => {
    if (!initialCode || submittedInitial.current) return;
    submittedInitial.current = true;
    void submitCode(initialCode);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once per arrival
  }, [initialCode]);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    void submitCode(input);
  };

  return (
    <Container size="xs" py="xl">
      <Stack gap="xl">
        <Group justify="flex-end">
          <LanguageSwitcher size="sm" variant="subtle" />
        </Group>
        <Card shadow="sm" padding="xl" radius="md" withBorder>
          <form onSubmit={handleSubmit}>
            <Stack gap="md">
              <Title order={2}>{t("enterCode.title")}</Title>
              <Text size="sm" c="dimmed">
                {t("enterCode.description")}
              </Text>
              <TextInput
                value={input}
                onChange={(e) => setInput(e.currentTarget.value)}
                placeholder={t("enterCode.placeholder")}
                size="lg"
                autoFocus
                autoComplete="off"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
                aria-label={t("enterCode.title")}
              />
              {error && (
                <Alert color="red" icon={<IconAlertCircle size={16} />}>
                  {error}
                </Alert>
              )}
              <ActionButton
                type="submit"
                loading={submitting}
                disabled={!input.trim()}
                rightSection={<IconArrowRight size={18} />}
              >
                {t("enterCode.submit")}
              </ActionButton>
            </Stack>
          </form>
        </Card>
      </Stack>
    </Container>
  );
}
