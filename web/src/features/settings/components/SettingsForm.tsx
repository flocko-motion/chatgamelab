import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Button,
  Card,
  Stack,
  TextInput,
  Title,
  Alert,
} from '@mantine/core';
import { IconUser, IconMail, IconCheck } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { useAuth } from '@/providers/AuthProvider';
import { Api } from '@/api/generated';
import { createAuthenticatedApiConfig } from '@/api/client/http';

export function SettingsForm() {
  const { t } = useTranslation('auth');
  const { backendUser, getAccessToken, retryBackendFetch } = useAuth();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState(false);

  const schema = z.object({
    name: z
      .string()
      .min(1, t('settings.errors.nameRequired'))
      .max(24, t('settings.errors.nameTooLong')),
    email: z
      .string()
      .min(1, t('settings.errors.emailRequired'))
      .email(t('settings.errors.emailInvalid')),
  });

  type FormData = z.infer<typeof schema>;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isDirty },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: backendUser?.name || '',
      email: backendUser?.email || '',
    },
  });

  // Reset form when backendUser changes
  useEffect(() => {
    if (backendUser) {
      reset({
        name: backendUser.name || '',
        email: backendUser.email || '',
      });
    }
  }, [backendUser, reset]);

  const onSubmit = async (data: FormData) => {
    if (!backendUser) return;

    setIsSubmitting(true);
    setSubmitError(null);
    setSubmitSuccess(false);

    try {
      const api = new Api(createAuthenticatedApiConfig(getAccessToken));
      await api.users.usersCreate(backendUser.id!, {
        name: data.name.trim(),
        email: data.email.trim(),
      });
      
      setSubmitSuccess(true);
      // Refresh backend user data
      retryBackendFetch();
      
      // Clear success message after 3 seconds
      setTimeout(() => setSubmitSuccess(false), 3000);
    } catch (error: unknown) {
      console.error('Failed to update settings:', error);
      
      // Check for specific error types
      if (error && typeof error === 'object' && 'error' in error) {
        const errorData = error as { error?: { message?: string } };
        const message = errorData.error?.message || '';
        
        if (message.includes('Name is already taken')) {
          setSubmitError(t('settings.errors.nameTaken'));
        } else if (message.includes('Email is already taken')) {
          setSubmitError(t('settings.errors.emailTaken'));
        } else {
          setSubmitError(t('settings.errors.saveFailed'));
        }
      } else {
        setSubmitError(t('settings.errors.saveFailed'));
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!backendUser) {
    return null;
  }

  return (
    <Card shadow="sm" padding="xl" radius="md" withBorder>
      <Stack gap="lg">
        <Title order={3}>{t('settings.accountSection')}</Title>

        <form onSubmit={handleSubmit(onSubmit)}>
          <Stack gap="md">
            <TextInput
              label={t('settings.nameLabel')}
              placeholder={t('settings.namePlaceholder')}
              description={t('settings.nameDescription')}
              leftSection={<IconUser size={16} />}
              error={errors.name?.message}
              {...register('name')}
              disabled={isSubmitting}
            />

            <TextInput
              label={t('settings.emailLabel')}
              placeholder={t('settings.emailPlaceholder')}
              description={t('settings.emailDescription')}
              leftSection={<IconMail size={16} />}
              error={errors.email?.message}
              {...register('email')}
              disabled={isSubmitting}
            />

            {submitError && (
              <Alert color="red" variant="light">
                {submitError}
              </Alert>
            )}

            {submitSuccess && (
              <Alert color="green" variant="light" icon={<IconCheck size={16} />}>
                {t('settings.saved')}
              </Alert>
            )}

            <Button
              type="submit"
              loading={isSubmitting}
              disabled={!isDirty}
              mt="sm"
            >
              {isSubmitting ? t('settings.saving') : t('settings.saveButton')}
            </Button>
          </Stack>
        </form>
      </Stack>
    </Card>
  );
}
