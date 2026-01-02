import { Button, Group, Paper, Text, Title } from '@mantine/core';

export function ErrorTest() {
  const throwSyncError = () => {
    throw new Error('This is a test synchronous error');
  };

  const throwAsyncError = async () => {
    throw new Error('This is a test asynchronous error');
  };

  const causeTypeError = () => {
    // @ts-ignore - Intentional type error for testing
    const obj: any = null;
    return obj.property.nested.property;
  };

  return (
    <Paper shadow="sm" p="md" withBorder>
      <Title order={3} mb="md">
        Error Boundary Test
      </Title>
      
      <Text mb="md" c="dimmed">
        These buttons will trigger different types of errors to test the ErrorBoundary component.
        Use this only in development mode for testing error handling.
      </Text>

      <Group>
        <Button
          color="red"
          onClick={throwSyncError}
          variant="outline"
        >
          Throw Sync Error
        </Button>
        
        <Button
          color="red"
          onClick={throwAsyncError}
          variant="outline"
        >
          Throw Async Error
        </Button>
        
        <Button
          color="orange"
          onClick={causeTypeError}
          variant="outline"
        >
          Cause Type Error
        </Button>
      </Group>
    </Paper>
  );
}
