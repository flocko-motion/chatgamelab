import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { Container, Group, Alert, Box } from '@mantine/core';
import { DangerButton } from './buttons';
import { SectionTitle, BodyText, HelperText } from './typography';
import { uiLogger } from '../../config/logger';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: ErrorInfo;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    // Update state to show error UI
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({
      error,
      errorInfo,
    });

    // Call custom error handler if provided
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // Log error to console in development
    if (import.meta.env.DEV) {
      uiLogger.error('ErrorBoundary caught an error', { error, errorInfo });
    }
  }

  handleReset = () => {
    this.setState({ hasError: false, error: undefined, errorInfo: undefined });
  };

  render() {
    if (this.state.hasError) {
      // Use custom fallback if provided, otherwise use default
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <Container size="md" py="xl">
          <div style={{ textAlign: 'center' }}>
            <Alert color="red" title="Error" icon="🚨">
              <SectionTitle>
                Something went wrong
              </SectionTitle>
              
              <BodyText>
                An unexpected error occurred. This has been logged and we'll look into it.
              </BodyText>

              {import.meta.env.DEV && this.state.error && (
                <Box
                  mb="md"
                  p="md"
                  style={{
                    background: 'var(--mantine-color-red-0)',
                    border: '1px solid var(--mantine-color-red-2)',
                    borderRadius: 'var(--mantine-radius-md)',
                    textAlign: 'left',
                  }}
                >
                  <HelperText>
                    Error Details (Development Mode):
                  </HelperText>
                  <Box
                    component="pre"
                    style={{
                      whiteSpace: 'pre-wrap',
                      fontFamily: 'monospace',
                      fontSize: '12px',
                      maxHeight: '200px',
                      overflow: 'auto',
                      margin: 0,
                      color: 'var(--mantine-color-red-7)',
                    }}
                  >
                    {this.state.error.toString()}
                    {this.state.errorInfo && this.state.errorInfo.componentStack}
                  </Box>
                </Box>
              )}

              <Group justify="center" gap="md">
                <DangerButton onClick={this.handleReset}>
                  Try Again
                </DangerButton>
                <DangerButton 
                  onClick={() => window.location.reload()} 
                  variant="outline"
                >
                  Reload Page
                </DangerButton>
              </Group>
            </Alert>
          </div>
        </Container>
      );
    }

    return this.props.children;
  }
}

// Export a wrapper that uses React's ErrorBoundary for better error catching
export function ErrorBoundaryWrapper({ children, fallback, onError }: Props) {
  return (
    <Component
      fallback={fallback}
      onError={(error: Error, errorInfo: ErrorInfo) => {
        uiLogger.error('Unhandled error', { error, errorInfo });
        if (onError) {
          onError(error, errorInfo);
        }
      }}
    >
      {children}
    </Component>
  );
}