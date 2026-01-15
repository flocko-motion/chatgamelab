import { useState, useEffect } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Box,
  Card,
  Container,
  Stack,
  Text,
  TextInput,
  Loader,
} from '@mantine/core';
import { ActionButton, TextButton } from '@components/buttons';
import { SectionTitle, HelperText } from '@components/typography';
import { IconUser, IconMail, IconCheck, IconX } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useDebouncedValue } from '@mantine/hooks';
import { authLogger } from '@/config/logger';

import { useAuth, type RegistrationData } from '@/providers/AuthProvider';
import { Api } from '@/api/generated';
import { createAuthenticatedApiConfig } from '@/api/client/http';

interface RegistrationFormProps {
  registrationData: RegistrationData;
  onCancel?: () => void;
}

export function RegistrationForm({ registrationData, onCancel }: RegistrationFormProps) {
  const { t } = useTranslation('auth');
  const navigate = useNavigate();
  const { register: registerUser, getAccessToken, logout } = useAuth();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const schema = z.object({
    name: z
      .string()
      .min(1, t('register.errors.nameRequired'))
      .max(24, t('register.errors.nameTooLong')),
    email: z
      .string()
      .min(1, t('register.errors.emailRequired'))
      .email(t('register.errors.emailInvalid')),
  });

  type FormData = z.infer<typeof schema>;

  const {
    register,
    handleSubmit,
    watch,
    setError,
    clearErrors,
    formState: { errors },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: registrationData.name || '',
      email: registrationData.email || '',
    },
  });

  const nameValue = watch('name');
  const [debouncedName] = useDebouncedValue(nameValue, 500);
  const [isCheckingName, setIsCheckingName] = useState(false);
  const [nameAvailable, setNameAvailable] = useState<boolean | null>(null);

  useEffect(() => {
    const checkNameAvailability = async () => {
      if (!debouncedName || debouncedName.length === 0 || debouncedName.length > 24) {
        setNameAvailable(null);
        return;
      }

      setIsCheckingName(true);
      try {
        const api = new Api(createAuthenticatedApiConfig(getAccessToken));
        const response = await api.auth.checkNameList({ name: debouncedName });
        const available = response.data.available ?? false;
        setNameAvailable(available);
        
        if (!available) {
          setError('name', { 
            type: 'manual', 
            message: t('register.errors.nameTaken') 
          });
        } else {
          clearErrors('name');
        }
      } catch (error) {
        authLogger.error('Failed to check name availability', { error });
        setNameAvailable(null);
      } finally {
        setIsCheckingName(false);
      }
    };

    checkNameAvailability();
  }, [debouncedName, getAccessToken, setError, clearErrors, t]);

  const onSubmit = async (data: FormData) => {
    if (nameAvailable === false) {
      return;
    }

    setIsSubmitting(true);
    setSubmitError(null);

    try {
      await registerUser(data.name.trim(), data.email.trim());
      navigate({ to: '/dashboard' });
    } catch (error) {
      authLogger.error('Registration failed', { error });
      setSubmitError(t('register.errors.registrationFailed'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    if (onCancel) {
      onCancel();
    } else {
      logout();
    }
  };

  const getNameRightSection = () => {
    if (isCheckingName) {
      return <Loader size="xs" />;
    }
    if (nameAvailable === true && !errors.name) {
      return <IconCheck size={16} color="green" />;
    }
    if (nameAvailable === false || errors.name) {
      return <IconX size={16} color="red" />;
    }
    return null;
  };

  return (
    <Container size="xs" py={{ base: 'md', sm: 'xl' }}>
      <Card shadow="md" padding="xl" radius="md" withBorder>
        <Stack gap="lg">
          <Box ta="center">
            <SectionTitle>
              {t('register.title')}
            </SectionTitle>
            <HelperText>
              {t('register.subtitle')}
            </HelperText>
          </Box>

          <form onSubmit={handleSubmit(onSubmit)}>
            <Stack gap="md">
              <TextInput
                label={t('register.nameLabel')}
                placeholder={t('register.namePlaceholder')}
                description={t('register.nameDescription')}
                leftSection={<IconUser size={16} />}
                rightSection={getNameRightSection()}
                error={errors.name?.message}
                {...register('name')}
                disabled={isSubmitting}
              />

              <TextInput
                label={t('register.emailLabel')}
                placeholder={t('register.emailPlaceholder')}
                description={t('register.emailDescription')}
                leftSection={<IconMail size={16} />}
                error={errors.email?.message}
                {...register('email')}
                disabled={isSubmitting}
              />

              {submitError && (
                <Text c="red" size="sm" ta="center">
                  {submitError}
                </Text>
              )}

              <Stack gap="xs" mt="md">
                <ActionButton
                  type="submit"
                  fullWidth
                  loading={isSubmitting}
                  disabled={isCheckingName || nameAvailable === false}
                >
                  {isSubmitting
                    ? t('register.submitting')
                    : t('register.submitButton')}
                </ActionButton>

                <TextButton
                  onClick={handleCancel}
                  disabled={isSubmitting}
                >
                  {t('register.cancelButton')}
                </TextButton>
              </Stack>
            </Stack>
          </form>
        </Stack>
      </Card>
    </Container>
  );
}
