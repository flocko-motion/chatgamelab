import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Card, Stack, TextInput, Alert, Switch } from "@mantine/core";
import { ActionButton } from "@components/buttons";
import { SectionTitle } from "@components/typography";
import { IconUser, IconMail, IconCheck } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

import { useAuth } from "@/providers/AuthProvider";
import { uiLogger } from "@/config/logger";
import { useUpdateUser } from "@/api/hooks";

export function SettingsForm() {
  const { t } = useTranslation("auth");
  const { backendUser, retryBackendFetch } = useAuth();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  const updateUser = useUpdateUser();

  const schema = z.object({
    name: z
      .string()
      .min(1, t("settings.errors.nameRequired"))
      .max(24, t("settings.errors.nameTooLong")),
    email: z
      .string()
      .min(1, t("settings.errors.emailRequired"))
      .email(t("settings.errors.emailInvalid")),
    showAiModelSelector: z.boolean(),
  });

  type FormData = z.infer<typeof schema>;

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors, isDirty },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: backendUser?.name || "",
      email: backendUser?.email || "",
      showAiModelSelector: backendUser?.showAiModelSelector || false,
    },
  });

  const showAiModelSelector = watch("showAiModelSelector");

  // Reset form when backendUser changes
  useEffect(() => {
    if (backendUser) {
      reset({
        name: backendUser.name || "",
        email: backendUser.email || "",
        showAiModelSelector: backendUser.showAiModelSelector || false,
      });
    }
  }, [backendUser, reset]);

  const onSubmit = (data: FormData) => {
    if (!backendUser) return;

    setSubmitError(null);
    setSubmitSuccess(false);

    updateUser.mutate(
      {
        id: backendUser.id!,
        request: {
          name: data.name.trim(),
          email: data.email.trim(),
          showAiModelSelector: data.showAiModelSelector,
        },
      },
      {
        onSuccess: () => {
          setSubmitSuccess(true);
          // Refresh backend user data in AuthProvider
          retryBackendFetch();
          // Clear success message after 3 seconds
          setTimeout(() => setSubmitSuccess(false), 3000);
        },
        onError: (error: unknown) => {
          uiLogger.error("Failed to update settings", { error });

          // Check for specific error types
          if (error && typeof error === "object" && "error" in error) {
            const errorData = error as { error?: { message?: string } };
            const message = errorData.error?.message || "";

            if (message.includes("Name is already taken")) {
              setSubmitError(t("settings.errors.nameTaken"));
            } else if (message.includes("Email is already taken")) {
              setSubmitError(t("settings.errors.emailTaken"));
            } else {
              setSubmitError(t("settings.errors.saveFailed"));
            }
          } else {
            setSubmitError(t("settings.errors.saveFailed"));
          }
        },
      },
    );
  };

  const isSubmitting = updateUser.isPending;

  if (!backendUser) {
    return null;
  }

  return (
    <Card shadow="sm" padding="xl" radius="md" withBorder>
      <Stack gap="lg">
        <SectionTitle>{t("settings.accountSection")}</SectionTitle>

        <form onSubmit={handleSubmit(onSubmit)}>
          <Stack gap="md">
            <TextInput
              label={t("settings.nameLabel")}
              placeholder={t("settings.namePlaceholder")}
              description={t("settings.nameDescription")}
              leftSection={<IconUser size={16} />}
              error={errors.name?.message}
              {...register("name")}
              disabled={isSubmitting}
            />

            <TextInput
              label={t("settings.emailLabel")}
              placeholder={t("settings.emailPlaceholder")}
              description={t("settings.emailDescription")}
              leftSection={<IconMail size={16} />}
              error={errors.email?.message}
              {...register("email")}
              disabled={isSubmitting}
            />

            <Stack gap="xs" mt="lg">
              <Switch
                label={t("settings.showAiModelSelectorLabel")}
                description={t("settings.showAiModelSelectorDescription")}
                checked={showAiModelSelector}
                onChange={(event) =>
                  setValue("showAiModelSelector", event.currentTarget.checked, {
                    shouldDirty: true,
                  })
                }
                disabled={isSubmitting}
              />
            </Stack>

            {submitError && (
              <Alert color="red" variant="light">
                {submitError}
              </Alert>
            )}

            {submitSuccess && (
              <Alert
                color="green"
                variant="light"
                icon={<IconCheck size={16} />}
              >
                {t("settings.saved")}
              </Alert>
            )}

            <ActionButton
              type="submit"
              loading={isSubmitting}
              disabled={!isDirty}
              size="md"
            >
              {isSubmitting ? t("settings.saving") : t("settings.saveButton")}
            </ActionButton>
          </Stack>
        </form>
      </Stack>
    </Card>
  );
}
