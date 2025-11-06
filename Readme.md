<div align="center">

# ObjectWeaver

[![License](https://img.shields.io/badge/License-EULA-blue.svg)](https://github.com/objectweaver/objectweaver/blob/main/LICENSE.txt)
[![Docker](https://img.shields.io/badge/Docker-Ready-brightgreen.svg)](https://hub.docker.com/r/objectweaver/objectweaver)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![Documentation](https://img.shields.io/badge/Docs-objectweaver.dev-orange.svg)](https://objectweaver.dev/docs)

[Website](https://objectweaver.dev) • [Documentation](https://objectweaver.dev/docs) • [API Reference](https://objectweaver.dev/api)

</div>

ObjectWeaver is an LLM orchestration service for generating structured objects in JSON format. Utilising the benefits of Go's concurrency to have parallel processing of requests to improve the speed at which fields are generated. Along with options to link fields, make decisions and have output validation baked into the processing engine. Allowing you to focus on the quality of your prompts and model selection and let the orchestration be handled by ObjectWeaver. 

For complete documentation, examples, and guides, visit [objectweaver.dev](https://objectweaver.dev).

## Getting Started

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

- [Full documentation](https://objectweaver.dev/docs)
- [API reference](https://objectweaver.dev/api)
- [Examples and tutorials](https://objectweaver.dev/examples)
- [Configuration guide](https://objectweaver.dev/docs/configuration)

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.MD) for the guidelines.

## License

See [LICENSE.txt](LICENSE.txt) for details.