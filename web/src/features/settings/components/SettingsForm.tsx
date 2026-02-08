import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Card, Stack, TextInput, Alert, Switch, Loader } from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import { ActionButton } from "@components/buttons";
import { SectionTitle } from "@components/typography";
import { IconUser, IconCheck, IconX, IconInfoCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

import { useAuth } from "@/providers/AuthProvider";
import { uiLogger } from "@/config/logger";
import { useUpdateUser } from "@/api/hooks";
import { Api } from "@/api/generated";
import { createAuthenticatedApiConfig } from "@/api/client/http";

export function SettingsForm() {
  const { t } = useTranslation("auth");
  const { backendUser, retryBackendFetch, getAccessToken } = useAuth();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  const updateUser = useUpdateUser();


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
    setError,
    clearErrors,
    formState: { errors, isDirty },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: backendUser?.name || "",
      showAiModelSelector: backendUser?.showAiModelSelector || false,
    },
  });

  const showAiModelSelector = watch("showAiModelSelector");
  const nameValue = watch("name");
  const [debouncedName] = useDebouncedValue(nameValue, 500);
  const [isCheckingName, setIsCheckingName] = useState(false);
  const [nameAvailable, setNameAvailable] = useState<boolean | null>(null);

  // Debounced name availability + profanity check
  useEffect(() => {
    const checkName = async () => {
      // Skip if name is empty, too long, or unchanged from current
      if (
        !debouncedName ||
        debouncedName.length === 0 ||
        debouncedName.length > 24 ||
        debouncedName === backendUser?.name
      ) {
        setNameAvailable(null);
        return;
      }

      setIsCheckingName(true);
      try {
        const api = new Api(createAuthenticatedApiConfig(getAccessToken));
        const response = await api.auth.checkNameList({ name: debouncedName });
        const data = response.data as { available?: boolean; profane?: boolean };
        const available = data.available ?? false;
        setNameAvailable(available);

        if (!available) {
          setError("name", {
            type: "manual",
            message: data.profane
              ? t("errors:nameProfane")
              : t("settings.errors.nameTaken"),
          });
        } else {
          clearErrors("name");
        }
      } catch (error) {
        uiLogger.error("Failed to check name availability", { error });
        setNameAvailable(null);
      } finally {
        setIsCheckingName(false);
      }
    };

    checkName();
  }, [debouncedName, backendUser?.name, getAccessToken, setError, clearErrors, t]);

  const getNameRightSection = () => {
    if (isCheckingName) return <Loader size="xs" />;
    if (nameAvailable === true && !errors.name) return <IconCheck size={16} color="green" />;
    if (nameAvailable === false || errors.name) return <IconX size={16} color="red" />;
    return null;
  };

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
    if (nameAvailable === false) return;

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
            const errorData = error as { error?: { data?: { code?: string }; message?: string } };
            const code = errorData.error?.data?.code || "";
            const message = errorData.error?.message || "";

            if (code === "profane_name") {
              setSubmitError(t("errors:nameProfane"));
            } else if (message.includes("Name is already taken")) {
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
              rightSection={getNameRightSection()}
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
              disabled={!isDirty || isCheckingName || nameAvailable === false}
              size="md"
            >
              {isSubmitting ? t("settings.saving") : t("settings.saveButton")}
            </ActionButton>
          </Stack>
        </form>
      </Stack>
    </Card>

    </Stack>
  );
}
