<div align="center">

# <span style="font-family: 'Roboto', sans-serif;">ObjectWeaver</span>

[![License](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://github.com/objectweaver/objectweaver/blob/main/LICENSE.txt)
[![Docker](https://img.shields.io/badge/Docker-Ready-brightgreen.svg)](https://hub.docker.com/r/objectweaver/objectweaver)
[![Documentation](https://img.shields.io/badge/Docs-objectweaver.dev-orange.svg)](https://objectweaver.dev/docs)

</div>

ObjectWeaver is a schema-first orchestration engine that sits between your application and your LLM providers. You describe a pipeline of tasks as a JSON schema: each field is a node in a task network with its own model, instructions, and dependencies. ObjectWeaver routes each field to the right model, processes independent fields in parallel, and wires the results together into a single JSON response.

For complete documentation, examples, and guides, visit [our documentation](https://objectweaver.dev/docs).

<div align="center">
  <img src="demo.gif" alt="ObjectWeaver Demo" width="800"/>
</div>

## Why ObjectWeaver?

Traditional LLM pipelines treat generation as a single monolithic call — one model, one prompt, all fields at once. Grammar-constrained alternatives guarantee syntax but force a one-size-fits-all approach. ObjectWeaver takes a different path: field-level orchestration where each part of your output is independently routed, validated, and assembled into a coherent result. With naive JSON generation success rates as low as [35-65%](https://composio.dev/blog/gpt-4-function-calling-example), that guarantee matters — but it's a natural outcome of the network model, not what defines it. The key advantages are:

- <img src="https://api.iconify.design/lucide/circle-check-big.svg?color=%23005221" width="16" height="16" style="vertical-align: text-bottom;" /> **Guaranteed JSON Output**: Compositional assembly validates each field independently, so the final response is always structurally sound — without inference-time constraints that degrade model reasoning.
- <img src="https://api.iconify.design/lucide/zap.svg?color=%23006329" width="16" height="16" style="vertical-align: text-bottom;" /> **Parallel Generation**: Independent nodes in the task network are processed concurrently, leading to significantly faster generation across production schemas with many fields.
- <img src="https://api.iconify.design/lucide/sparkles.svg?color=%23007431" width="16" height="16" style="vertical-align: text-bottom;" /> **Model Specialization**: Route simple tasks to efficient models and complex reasoning to more powerful ones, reducing costs by 10-20x.
- <img src="https://api.iconify.design/lucide/expand.svg?color=%2300943d" width="16" height="16" style="vertical-align: text-bottom;" /> **Break Context Limits**: Generate massive datasets and comprehensive documents that exceed the context window of a single model.
- <img src="https://api.iconify.design/lucide/bot.svg?color=%23005221" width="16" height="16" style="vertical-align: text-bottom;" /> **Composable Intelligence**: Treat your schema as a network of interrelated tasks. Fields feed context to downstream fields, branch on generated values, fetch external data, and wire results together — expressing complex multi-step workflows as a single JSON definition.

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

Here's how to make a basic API call. The `definition` field describes a task network: each property is a node with its own model, instruction, and optional dependencies on other fields.

### cURL

```bash
curl -X POST http://localhost:2008/api/objectGen \
  -H "Authorization: Bearer your-api-token" \
  -H "Content-Type: application/json" \
  -d '{
        "prompt": "Generate a schema that defines the technological landscape of the world",
        "definition": {
          "type": "object",
          "instruction": "Defines the technological landscape of the world, including its level of advancement and notable innovations.",
          "properties": {
            "Level": {
              "type": "string",
              "instruction": "Categorize the overall technological sophistication of the world, such as medieval, industrial, or advanced futuristic."
            },
            "Inventions": {
              "type": "string",
              "instruction": "Describe the most significant technological discoveries and their transformative impact on the society, economy, and daily life."
            }
          }
        }
      }
'
```

Find more different language examples [here](https://objectweaver.dev/docs/api-reference/curl-examples).

## Features

ObjectWeaver's task-network model unlocks capabilities that go beyond what any single LLM call can provide:

-   <img src="https://api.iconify.design/lucide/git-branch.svg?color=%23006329" width="16" height="16" style="vertical-align: text-bottom;" /> **Decision Points**: Embed adaptive intelligence in your schemas to dynamically alter the generation process based on the output of other fields.
-   <img src="https://api.iconify.design/lucide/shield-check.svg?color=%2300943d" width="16" height="16" style="vertical-align: text-bottom;" /> **Epistemic Validation**: Implement validation and retry logic to ensure the quality and accuracy of the generated data.
-   <img src="https://api.iconify.design/lucide/download.svg?color=%23005221" width="16" height="16" style="vertical-align: text-bottom;" /> **Data Fetching**: Fetch data from external sources and use it as context for generation.
-   <img src="https://api.iconify.design/lucide/radio.svg?color=%23007431" width="16" height="16" style="vertical-align: text-bottom;" /> **Streaming Requests**: Stream data as it's generated for real-time applications.
-   <img src="https://api.iconify.design/lucide/layers.svg?color=%23005221" width="16" height="16" style="vertical-align: text-bottom;" /> **Batch Processing & Priority System**: Optimize costs by up to 50% by routing non-urgent requests to OpenAI's Batch API. You can assign priorities to different fields to balance speed and cost.

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

For a full list of configuration options and what they do, check out the [configuration guide](https://objectweaver.dev/docs/getting-started/docker-setup).

## Building from Source

If you prefer to build ObjectWeaver yourself:

```bash
git clone https://github.com/objectweaver/objectweaver.git
cd objectweaver
go build -o objectweaver .
./objectweaver
```

You're also able to find the compiled binaries in the [releases](https://github.com/ObjectWeaver/ObjectWeaver/releases).

## Community and Support

-   <img src="https://api.iconify.design/lucide/star.svg?color=%23005221" width="16" height="16" style="vertical-align: text-bottom;" /> **Star on GitHub**: If you find ObjectWeaver useful, please [give us a star on GitHub](https://github.com/objectweaver/objectweaver)! It helps others discover the project.
-   **Documentation**: For detailed guides, examples, and API references, visit our [documentation website](https://objectweaver.dev/docs).
-   **GitHub Issues**: If you encounter a bug or have a feature request, please [open an issue on GitHub](https://github.com/objectweaver/objectweaver/issues).
-   **Contact Us**: For enterprise inquiries, please [contact us](https://objectweaver.dev/enterprise).

## Contributing

Contributions are welcome! Please see our [CONTRIBUTING guide](https://github.com/ObjectWeaver/ObjectWeaver/blob/main/CONTRIBUTING.MD) for guidelines on how to contribute to the project.

## License

ObjectWeaver uses a **dual licensing model**:

### Community Edition (AGPL-3)

The ObjectWeaver Community Edition is available under the GNU Affero General Public License v3. It is free for internal tools, development, and open-source projects. There are no restrictions on self-hosted deployments within your organization. However, if you offer it as a network service to third parties (e.g. SaaS), you must make your modified source code available under the same license.

### Commercial License

Building a SaaS product or proprietary service? The Commercial License removes open-source obligations and includes enterprise-grade support, legal protection, and compliance assistance. The code in the `ee/` directory is licensed under this commercial license.

For commercial licensing inquiries, visit [our enterprise page](https://objectweaver.dev/enterprise) or contact enterprise@objectweaver.dev. 
