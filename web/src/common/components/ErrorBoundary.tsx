import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { Button, Container, Group, Text, Title } from '@mantine/core';

interface Props {
  children: ReactNode;
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
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({
      error,
      errorInfo,
    });

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
      return (
        <Container size="md" py="xl">
          <div style={{ textAlign: 'center' }}>
            <Title order={1} c="red" mb="md">
              Something went wrong
            </Title>
            
            <Text mb="lg">
              An unexpected error occurred. This has been logged and we'll look into it.
            </Text>

            {import.meta.env.DEV && this.state.error && (
              <div style={{ 
                background: '#f5f5f5', 
                padding: '1rem', 
                borderRadius: '4px', 
                textAlign: 'left',
                marginBottom: '1rem'
              }}>
                <Text size="sm" fw={500} mb="xs">
                  Error Details (Development Mode):
                </Text>
                <Text size="xs" component="pre" style={{ whiteSpace: 'pre-wrap' }}>
                  {this.state.error.toString()}
                  {this.state.errorInfo && this.state.errorInfo.componentStack}
                </Text>
              </div>
            )}

            <Group justify="center">
              <Button onClick={this.handleReset} variant="filled">
                Try Again
              </Button>
              <Button 
                onClick={() => window.location.reload()} 
                variant="outline"
              >
                Reload Page
              </Button>
            </Group>
          </div>
        </Container>
      );
    }

    return this.props.children;
  }
}