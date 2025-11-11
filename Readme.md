<div align="center">

# ObjectWeaver

[![License](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://github.com/objectweaver/objectweaver/blob/main/LICENSE.txt)
[![Docker](https://img.shields.io/badge/Docker-Ready-brightgreen.svg)](https://hub.docker.com/r/objectweaver/objectweaver)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![Documentation](https://img.shields.io/badge/Docs-objectweaver.dev-orange.svg)](https://objectweaver.dev/docs)

[Website](https://objectweaver.dev) • [Documentation](https://objectweaver.dev/docs) • [API Reference](https://objectweaver.dev/api)

</div>

ObjectWeaver is an AI Orchestration Service for generating structured objects in JSON format. It guarantees 100% valid JSON output by decomposing schemas into field-level tasks, routing them to optimal language models, and processing them in parallel. This approach not only ensures reliability but also significantly reduces costs and improves performance by using the best model for each task.

For complete documentation, examples, and guides, visit [objectweaver.dev](https://objectweaver.dev).

## Table of Contents

- Why ObjectWeaver?
- Getting Started
- Making Your First Request
- Features
- Configuration
- Building from Source
- Community and Support
- Contributing
- License

## Why ObjectWeaver?

Traditional JSON generation with LLMs often fails, with success rates as low as 35-65%. While grammar-constrained alternatives can guarantee syntax, they force a one-size-fits-all approach, using a single model and prompt for all fields. ObjectWeaver solves this by providing intelligent, field-level orchestration that offers several key advantages:

- **Guaranteed JSON Output**: Field-level type validation and compositional assembly ensure 100% valid JSON every time.
- **Parallel Generation**: Independent fields are generated concurrently, leading to significantly faster processing times.
- **Model Specialization**: Route simple tasks to efficient models and complex reasoning to more powerful ones, reducing costs by 10-20x.
- **Break Context Limits**: Generate massive datasets and comprehensive documents that exceed the context window of a single model.
- **Field Dependencies**: Create complex workflows where the output of one field can be used as input for another.

## Getting Started

The easiest way to get ObjectWeaver running is with Docker.

1.  **Pull the Docker image:**

    ```bash
    docker pull objectweaver/objectweaver:latest
    ```

2.  **Run the Docker container:**

    ```bash
    docker run -p 2008:2008 \
      -e PASSWORD=your-request-api-key \
      -e OPENAI_API_KEY=your-openai-key \
      objectweaver/objectweaver:latest
    ```

    - `PASSWORD`: Your chosen API key for securing the ObjectWeaver service.
    - `OPENAI_API_KEY`: Your OpenAI API key. ObjectWeaver can also be configured to use other providers like Gemini or a local model.

That's it! The server will be running on `localhost:2008`.

## Making Your First Request

Here’s how to make a basic API call to generate a structured JSON object. The `definition` field uses standard JSON Schema syntax to specify the desired output structure.

### cURL

```bash
curl -X POST http://localhost:2008/api/objectGen \
  -H "Authorization: Bearer your-api-token" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Generate a user profile for a software engineer",
    "definition": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "email": { "type": "string" },
        "skills": { "type": "array", "items": { "type": "string" } },
        "experience_years": { "type": "integer" }
      }
    }
  }'
```

### Python

```python
import requests
import json

url = "http://localhost:2008/api/objectGen"
headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer your-api-token"
}

data = {
    "prompt": "Generate a user profile for a software engineer",
    "definition": {
        "type": "object",
        "properties": {
            "name": {"type": "string"},
            "email": {"type": "string"},
            "skills": {"type": "array", "items": {"type": "string"}},
            "experience_years": {"type": "integer"}
        }
    }
}

response = requests.post(url, headers=headers, json=data)
print(response.json())
```

### JavaScript (Node.js)

```javascript
const fetch = require('node-fetch');

const url = 'http://localhost:2008/api/objectGen';
const headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-api-token'
};

const data = {
    prompt: 'Generate a user profile for a software engineer',
    definition: {
        type: 'object',
        properties: {
            name: { type: 'string' },
            email: { type: 'string' },
            skills: { type: 'array', items: { type: 'string' } },
            experience_years: { type: 'integer' }
        }
    }
};

fetch(url, {
    method: 'POST',
    headers: headers,
    body: JSON.stringify(data)
})
.then(response => response.json())
.then(data => console.log(data))
.catch(error => console.error('Error:', error));
```

You'll get back a JSON response with your generated object and the cost:

```json
{
  "data": {
    "name": "Alex Johnson",
    "email": "alex.johnson@example.com",
    "skills": ["Go", "Kubernetes", "PostgreSQL", "gRPC"],
    "experience_years": 7
  },
  "usdCost": 0.0023
}
```

## Features

ObjectWeaver is designed for production use and includes several powerful features to handle real-world complexity:

-   **Batch Processing & Priority System**: Optimize costs by up to 50% by routing non-urgent requests to OpenAI's Batch API. You can assign priorities to different fields to balance speed and cost.
-   **Decision Points**: Embed adaptive intelligence in your schemas to dynamically alter the generation process based on the output of other fields.
-   **Streaming Requests**: Stream data as it's generated for real-time applications.
-   **Quality Assurance**: Implement validation and retry logic to ensure the quality and accuracy of the generated data.
-   **Data Fetching**: Fetch data from external sources and use it as context for generation.
-   **Prometheus Metrics**: Track costs, performance, and other key metrics with built-in Prometheus support.

## Configuration

Configure the service using environment variables. Here are some of the main options:

```bash
# Required for API access
PASSWORD=your-secure-api-token
# Your LLM provider key
OPENAI_API_KEY=your-openai-api-key

# LLM provider settings
LLM_PROVIDER=openai                      # openai, gemini, or local
LLM_API_URL=https://api.openai.com/v1
LLM_MAX_TOKENS_PER_MINUTE=150000
LLM_MAX_REQUESTS_PER_MINUTE=500

# Server settings
PORT=2008
ENVIRONMENT=production
```

For a full list of configuration options and what they do, check out the [configuration guide](https://objectweaver.dev/docs/configuration).

## Building from Source

If you prefer to build ObjectWeaver yourself:

```bash
git clone https://github.com/objectweaver/objectweaver.git
cd objectweaver
go build -o objectweaver .
./objectweaver
```

## Community and Support

-   **Documentation**: For detailed guides, examples, and API references, visit our [documentation website](https://objectweaver.dev/docs).
-   **GitHub Issues**: If you encounter a bug or have a feature request, please [open an issue on GitHub](https://github.com/objectweaver/objectweaver/issues).
-   **Contact Us**: For enterprise inquiries, please [contact us](https://objectweaver.dev/contact).

## Contributing

Contributions are welcome! Please see CONTRIBUTING.md for guidelines on how to contribute to the project.

## License

ObjectWeaver uses a **dual licensing model**:

### Community Edition (AGPL-3)

The ObjectWeaver Community Edition is available under the GNU Affero General Public License v3. This means it is free to use, modify, and distribute, but if you offer it as a network service, you must make your modified source code available under the same license.

### Enterprise Edition

The code in the `ee/` directory is licensed under the ObjectWeaver Commercial License and requires a valid ObjectWeaver Enterprise Edition subscription for production use. This edition includes advanced features such as SSO, multi-tenancy, and enhanced monitoring.

For more details, see LICENSE.txt or contact us for [Enterprise Edition inquiries](https://objectweaver.dev/contact).## Getting Started

Easiest way to get ObjectWeaver running is with Docker:

```bash
docker pull objectweaver/objectweaver:latest

docker run -p 2008:2008 \
  -e PASSWORD=your-request-api-key \
  -e OPENAI_API_KEY=your-openai-key \
  objectweaver/objectweaver:latest
```

That's it. Server will be running on `localhost:2008`.

## Making Your First Request

Here's what a basic API call looks like:

```bash
curl -X POST http://localhost:2008/api/objectGen \
  -H "Authorization: Bearer your-api-token" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Generate a user profile for a software engineer",
    "definition": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "email": { "type": "string" },
        "skills": { "type": "array", "items": { "type": "string" } },
        "experience_years": { "type": "integer" }
      }
    }
  }'
```

You'll get back a JSON response with your generated object and the cost:

```json
{
  "data": {
    "name": "Alex Johnson",
    "email": "alex.johnson@example.com",
    "skills": ["Go", "Kubernetes", "PostgreSQL", "gRPC"],
    "experience_years": 7
  },
  "usdCost": 0.0023
}
```

The `definition` field uses standard JSON Schema syntax to specify what structure you want back.

## Configuration

Need more control? Configure the service using environment variables. Main ones are:

```bash
PASSWORD=your-secure-api-token           # Required for API access
OPENAI_API_KEY=your-openai-api-key      # Or your LLM provider key

LLM_PROVIDER=openai                      # openai, gemini, or local
LLM_API_URL=https://api.openai.com/v1
LLM_MAX_TOKENS_PER_MINUTE=150000
LLM_MAX_REQUESTS_PER_MINUTE=500

PORT=2008
ENVIRONMENT=production
```

For a full list of configuration options and what they do, check out the [configuration guide](https://objectweaver.dev/docs/configuration).

## Building from Source

Prefer to build it yourself?

```bash
git clone https://github.com/objectweaver/objectweaver.git
cd objectweaver
go build -o objectweaver .
./objectweaver
```

## What Makes This Useful

Most LLM wrappers are pretty basic. ObjectWeaver handles the annoying parts of production usage:

- Validates the LLM actually returns the structure you asked for
- Retries failed requests with exponential backoff
- Handles rate limits across multiple providers (OpenAI, Gemini, local models)
- Gives you both REST and gRPC interfaces
- Tracks costs and exposes Prometheus metrics

It's designed to be the boring infrastructure piece you don't want to build yourself.

## Documentation

Everything else you need is on the website:

- [High Level documentation](https://objectweaver.dev/docs)
- [API reference](https://objectweaver.dev/api)
- [Examples and tutorials](https://objectweaver.dev/examples)
- [Configuration guide](https://objectweaver.dev/docs/configuration)
- Low level documentation using CodeLeft

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.MD) for the guidelines.

## License

ObjectWeaver uses a **dual licensing model**:

### Community Edition (AGPL-3)
ObjectWeaver Community Edition is available under the [GNU Affero General Public License v3](LICENSE.txt). This means:
- **Free to use**: Use ObjectWeaver for any purpose at no cost
- **Modify freely**: Change the code to suit your needs
- **Run as a service**: Deploy it commercially or non-commercially
- **Source code disclosure**: If you modify ObjectWeaver and offer it as a network service, you must make your modified source code available under AGPL-3
- **Copyleft**: Derivative works must also be licensed under AGPL-3

### Enterprise Edition (Source Available)
Code in the `ee/` directory is licensed under the [ObjectWeaver Commercial License](ee/LICENSE). This code is:
- **Source available**: You can view and audit the code
- **Requires subscription**: Production use requires a valid ObjectWeaver Enterprise Edition subscription
- **Additional features**: Advanced monitoring, SSO, multi-tenancy, and other enterprise capabilities

A commercial license is available. [Contact us](https://objectweaver.dev/contact) for Enterprise Edition inquiries.

### Licensing Summary
- **Core functionality** (outside `ee/` directory): AGPL-3
- **Enterprise features** (`ee/` directory): ObjectWeaver Commercial License
- **Third-party components**: Original licenses apply

See [LICENSE.txt](LICENSE.txt) for complete terms.

**Need help choosing?** If you're building an open-source project or can comply with AGPL-3 terms, use the Community Edition. If you need enterprise features or proprietary/closed-source usage, contact us about Enterprise Edition.