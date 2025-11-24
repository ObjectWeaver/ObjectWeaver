# Postman Tests

This directory contains the message payloads in the `messages/` folder for testing the server.

## Setup

### Authentication

Set the following metadata header for authentication:

```
x-api-key: YOUR_PASSWORD
```

### Prerequisites

- Ensure all required API keys are configured
- The service will not function without proper key configuration

## Usage

### Request Format

These messages are formatted for gRPC requests. They may work with other request types with appropriate modifications to the setup.

### Proto File

To use these tests, you'll need the protocol buffer definition:

- Use the proto file from the parent repository
OR
- Enable server reflection to retrieve the proto definition automatically 
