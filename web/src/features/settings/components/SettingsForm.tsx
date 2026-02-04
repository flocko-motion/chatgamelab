import { useState, useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Card, Stack, TextInput, Alert, Switch, Select } from "@mantine/core";
import { ActionButton } from "@components/buttons";
import { SectionTitle } from "@components/typography";
import { IconUser, IconCheck, IconInfoCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

import { useAuth } from "@/providers/AuthProvider";
import { uiLogger } from "@/config/logger";
import { useUpdateUser, usePlatforms, useSystemSettings, useUpdateSystemSettings } from "@/api/hooks";
import { isAdmin } from "@/common/lib/roles";

export function SettingsForm() {
  const { t } = useTranslation("auth");
  const { backendUser, retryBackendFetch } = useAuth();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  const updateUser = useUpdateUser();

  // Admin-only: system settings
  const userIsAdmin = backendUser ? isAdmin(backendUser) : false;
  const { data: platforms } = usePlatforms();
  const { data: systemSettings } = useSystemSettings();
  const updateSystemSettings = useUpdateSystemSettings();
  const [selectedDefaultModel, setSelectedDefaultModel] = useState<string | null>(null);
  const [adminSubmitSuccess, setAdminSubmitSuccess] = useState(false);
  const [adminSubmitError, setAdminSubmitError] = useState<string | null>(null);

  // Build model options from all platforms (grouped format for Mantine Select)
  const modelOptions = useMemo(() => {
    if (!platforms) return [];
    return platforms.map((platform) => ({
      group: platform.name || platform.id || "",
      items: (platform.models || []).map((model) => ({
        value: model.id || "",
        label: model.name || model.id || "",
      })),
    }));
  }, [platforms]);

  // Initialize selected model from system settings
  useEffect(() => {
    if (systemSettings?.defaultAiModel && !selectedDefaultModel) {
      setSelectedDefaultModel(systemSettings.defaultAiModel);
    }
  }, [systemSettings, selectedDefaultModel]);

  const schema = z.object({
    name: z
      .string()
      .min(1, t("settings.errors.nameRequired"))
      .max(24, t("settings.errors.nameTooLong")),
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
      showAiModelSelector: backendUser?.showAiModelSelector || false,
    },
  });

  const showAiModelSelector = watch("showAiModelSelector");

  // Reset form when backendUser changes
  useEffect(() => {
    if (backendUser) {
      reset({
        name: backendUser.name || "",
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

  const handleAdminSettingsSave = () => {
    if (!selectedDefaultModel) return;

    setAdminSubmitError(null);
    setAdminSubmitSuccess(false);

    updateSystemSettings.mutate(
      { defaultAiModel: selectedDefaultModel },
      {
        onSuccess: () => {
          setAdminSubmitSuccess(true);
          setTimeout(() => setAdminSubmitSuccess(false), 3000);
        },
        onError: (error: unknown) => {
          uiLogger.error("Failed to update system settings", { error });
          setAdminSubmitError(t("settings.errors.saveFailed"));
        },
      }
    );
  };

  const isAdminSettingsDirty = selectedDefaultModel !== systemSettings?.defaultAiModel;

  if (!backendUser) {
    return null;
  }

  return (
    <Stack gap="xl">
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

            <Alert
              variant="light"
              color="gray"
              icon={<IconInfoCircle size={16} />}
            >
              {t("settings.emailChangeNote")}
            </Alert>

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

    {/* Admin Settings Section */}
    {userIsAdmin && (
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Stack gap="lg">
          <SectionTitle>
            {t("settings.adminSection", "System Settings")}
          </SectionTitle>

          <Select
            label={t("settings.defaultAiModelLabel", "Default AI Model")}
            description={t("settings.defaultAiModelDescription", "The default AI model used when users don't select a specific model")}
            placeholder={t("settings.defaultAiModelPlaceholder", "Select a model")}
            data={modelOptions}
            value={selectedDefaultModel}
            onChange={setSelectedDefaultModel}
            searchable
            disabled={updateSystemSettings.isPending}
          />

          {adminSubmitError && (
            <Alert color="red" variant="light">
              {adminSubmitError}
            </Alert>
          )}

          {adminSubmitSuccess && (
            <Alert
              color="green"
              variant="light"
              icon={<IconCheck size={16} />}
            >
              {t("settings.saved")}
            </Alert>
          )}

          <ActionButton
            onClick={handleAdminSettingsSave}
            loading={updateSystemSettings.isPending}
            disabled={!isAdminSettingsDirty}
            size="md"
          >
            {updateSystemSettings.isPending ? t("settings.saving") : t("settings.saveButton")}
          </ActionButton>
        </Stack>
      </Card>
    )}
    </Stack>
  );
}
