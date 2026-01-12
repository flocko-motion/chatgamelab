import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { Button, Container, Group, Text, Title, Alert } from '@mantine/core';

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
      console.error('ErrorBoundary caught an error:', error, errorInfo);
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
              <Title order={2} c="red" mb="md">
                Something went wrong
              </Title>
              
              <Text mb="lg">
                An unexpected error occurred. This has been logged and we'll look into it.
              </Text>

              {import.meta.env.DEV && this.state.error && (
                <div style={{ 
                  background: '#fef2f2', 
                  border: '1px solid #fecaca',
                  padding: '1rem', 
                  borderRadius: '8px', 
                  textAlign: 'left',
                  marginBottom: '1rem'
                }}>
                  <Text size="sm" fw={500} mb="xs" c="red">
                    Error Details (Development Mode):
                  </Text>
                  <Text size="xs" component="pre" style={{ 
                    whiteSpace: 'pre-wrap', 
                    fontFamily: 'monospace',
                    fontSize: '12px',
                    maxHeight: '200px',
                    overflow: 'auto'
                  }}>
                    {this.state.error.toString()}
                    {this.state.errorInfo && this.state.errorInfo.componentStack}
                  </Text>
                </div>
              )}

              <Group justify="center" gap="md">
                <Button onClick={this.handleReset} variant="filled" color="red">
                  Try Again
                </Button>
                <Button 
                  onClick={() => window.location.reload()} 
                  variant="outline"
                  color="red"
                >
                  Reload Page
                </Button>
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
        console.error('Unhandled error:', error, errorInfo);
        if (onError) {
          onError(error, errorInfo);
        }
      }}
    >
      {children}
    </Component>
  );
}