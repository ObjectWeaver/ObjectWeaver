<div align="center">

# ObjectWeaver

**The Open-Source JSON Orchestration Service**

[![License](https://img.shields.io/badge/License-EULA-blue.svg)](https://github.com/objectweaver/objectweaver/blob/main/LICENSE.txt)
[![Docker](https://img.shields.io/badge/Docker-Ready-brightgreen.svg)](https://hub.docker.com/r/objectweaver/objectweaver)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![Documentation](https://img.shields.io/badge/Docs-objectweaver.dev-orange.svg)](https://objectweaver.dev/docs)

[Website](https://objectweaver.dev) • [Documentation](https://objectweaver.dev/)

</div>

---

## 📖 Overview

ObjectWeaver is a high-performance, production-ready object generation engine that leverages Large Language Models (LLMs) to transform natural language prompts into structured, validated data objects. Built with Go for speed and reliability, objectweaver provides both HTTP and gRPC interfaces for seamless integration into any application architecture.

### Key Features

- 🚀 **High Performance**: Built with Go for exceptional speed and low latency
- 🔌 **Dual Protocol Support**: HTTP REST API and gRPC for maximum flexibility  
- 🤖 **Multi-Provider LLM Support**: OpenAI, Gemini, local models, and custom providers
- 📊 **JSON Schema Validation**: Ensure generated objects match your exact specifications
- 🔄 **Streaming Support**: Real-time object generation with progressive updates
- 📈 **Production Observability**: Built-in Prometheus metrics and structured logging
- 🛡️ **Enterprise Security**: Token-based authentication, CORS, and rate limiting
- 🐳 **Docker Ready**: One-command deployment with docker-compose
- ⚡ **Backoff & Retry Logic**: Intelligent error handling and API rate limit management
- 🎯 **Plugin Architecture**: Extensible with pre/post-processors and validators

---

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose (recommended)
- Or: Go 1.21+ for building from source

### Installation

#### Option 1: Docker (Recommended)

```bash
# Pull the latest image
docker pull objectweaver/objectweaver:latest

# Or build locally
git clone https://github.com/objectweaver/objectweaver.git
cd objectweaver
docker-compose up -d
```

#### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/objectweaver/objectweaver.git
cd objectweaver

# Build the binary
go build -o objectweaver .

# Run the server
./objectweaver
```

### Configuration

Create a `.env` file in the project root:

```bash
# Required
PASSWORD=your-secure-api-token
OPENAI_API_KEY=your-openai-api-key

# LLM Provider Configuration
LLM_PROVIDER=openai  # or: gemini, local, custom
LLM_API_URL=https://api.openai.com/v1
LLM_API_KEY=your-api-key
LLM_MAX_TOKENS_PER_MINUTE=150000
LLM_MAX_REQUESTS_PER_MINUTE=500
LLM_BACKOFF_STRATEGY=exponential
LLM_USE_GZIP=true

# Optional
PORT=2008
ENVIRONMENT=production
VERBOSE_LOGS=false
GRPC_UNSECURE=false
```

### Your First Request

```bash
curl -X POST http://localhost:2008/api/objectGen \
  -H "Authorization: Bearer your-secure-api-token" \
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

**Response:**

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

---

## 📚 Architecture

objectweaver is architected for production-grade reliability and extensibility:

```
┌─────────────────────────────────────────────────┐
│              Client Applications                │
└─────────────┬───────────────────┬───────────────┘
              │                   │
         HTTP/REST              gRPC
              │                   │
┌─────────────┴───────────────────┴───────────────┐
│         Middleware Layer                        │
│  • CORS  • Auth  • Rate Limit  • Compression    │
└─────────────┬───────────────────────────────────┘
              │
┌─────────────┴───────────────────────────────────┐
│       Object Generation Orchestrator            │
│  • Schema Analysis  • Task Planning             │
│  • Execution Strategy  • Result Assembly        │
└─────────────┬───────────────────────────────────┘
              │
┌─────────────┴───────────────────────────────────┐
│         LLM Provider Management                 │
│  • OpenAI  • Gemini  • Local Models             │
│  • Backoff & Retry  • Token Management          │
└─────────────┬───────────────────────────────────┘
              │
         External LLM APIs
```

### Core Components

- **HTTP/gRPC Servers**: Dual-protocol support using chi router and native gRPC
- **Request Pipeline**: Middleware stack for auth, validation, CORS, compression
- **Generation Engine**: Schema-driven object generation with plugin support
- **LLM Client Manager**: Provider abstraction with intelligent retry and backoff
- **Model Converter**: Translate between internal and provider-specific formats
- **Observability**: Prometheus metrics, structured logging, request tracing

---

## 🔧 Advanced Usage

### Custom JSON Schemas

```json
{
  "prompt": "Create a product inventory item",
  "definition": {
    "type": "object",
    "properties": {
      "sku": { "type": "string", "pattern": "^[A-Z]{3}-\\d{4}$" },
      "name": { "type": "string" },
      "price": { "type": "number", "minimum": 0 },
      "stock": { "type": "integer" },
      "categories": {
        "type": "array",
        "items": { "type": "string" }
      },
      "metadata": {
        "type": "object",
        "properties": {
          "manufacturer": { "type": "string" },
          "warranty_months": { "type": "integer" }
        }
      }
    },
    "required": ["sku", "name", "price"]
  }
}
```

### gRPC Integration

```go
import (
    "context"
    "google.golang.org/grpc"
    pb "github.com/objectweaver/objectweaver/grpc"
)

// Connect to objectweaver gRPC server
conn, _ := grpc.Dial("localhost:2008", grpc.WithInsecure())
client := pb.NewObjectGenerationClient(conn)

// Generate object
resp, _ := client.GenerateObject(context.Background(), &pb.GenerateObjectRequest{
    Prompt: "Generate a blog post outline",
    Schema: &pb.Schema{
        Type: "object",
        Properties: map[string]*pb.Property{
            "title": {Type: "string"},
            "sections": {
                Type: "array",
                Items: &pb.Schema{Type: "string"},
            },
        },
    },
})
```

### Streaming Generation

```bash
curl -X POST http://localhost:2008/api/objectGen \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Generate a story",
    "stream": true,
    "definition": { ... }
  }'
```

---

## 🐳 Docker Deployment

### Using Docker Compose

```yaml
version: '3.8'

services:
  objectweaver:
    image: objectweaver/objectweaver:latest
    ports:
      - "2008:2008"
    environment:
      - PASSWORD=${PASSWORD}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - LLM_PROVIDER=openai
      - ENVIRONMENT=production
    networks:
      - objectweaver-network
    restart: unless-stopped

networks:
  objectweaver-network:
```

---

## 📊 Monitoring & Observability

objectweaver exposes Prometheus metrics at `/metrics`:

```bash
# Request metrics
http_requests_total{method="POST",path="/api/objectGen",status="200"}
http_request_duration_seconds{method="POST",path="/api/objectGen"}

# LLM provider metrics
llm_requests_total{provider="openai",model="gpt-4"}
llm_token_usage{provider="openai",type="prompt"}
llm_cost_usd{provider="openai"}

# System metrics
go_goroutines
process_cpu_seconds_total
process_resident_memory_bytes
```


---

### Contributing

We welcome contributions! Please see CONTRIBUTING.md for guidelines.

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Push to branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

---

## 🌟 Why objectweaver?

### Production-Ready

Unlike simple LLM wrappers, objectweaver is built for production workloads with:
- Robust error handling and retry logic
- Rate limiting and backoff strategies
- Comprehensive monitoring and metrics
- Schema validation and type safety

### Provider-Agnostic

Switch between OpenAI, Gemini, local (custom) models, or custom providers without changing your code. objectweaver abstracts provider-specific details while preserving advanced features.

### Performance-Focused

Written in Go for exceptional performance:
- Low latency request processing
- Efficient memory usage
- Concurrent request handling
- Gzip compression support

### Developer-Friendly

- Clear, idiomatic Go codebase
- Comprehensive documentation
- Type-safe gRPC and HTTP APIs
- Extensive examples and guides

---

## 🔗 Resources

- **Website**: [objectweaver.dev](https://objectweaver.dev)
- **Documentation**: [objectweaver.dev/docs](https://objectweaver.dev/docs)
- **API Reference**: [objectweaver.dev/api](https://objectweaver.dev/api)
- **Blog**: [objectweaver.dev/blog](https://objectweaver.dev/blog)
- **Issue Tracker**: [GitHub Issues](https://github.com/objectweaver/objectweaver/issues)

---

## 🙏 Acknowledgments

objectweaver is built with amazing open-source projects:

- [chi](https://github.com/go-chi/chi) - Lightweight HTTP router
- [go-openai](https://github.com/sashabaranov/go-openai) - OpenAI API client
- [gRPC](https://grpc.io/) - High-performance RPC framework
- [Prometheus](https://prometheus.io/) - Monitoring and alerting

---

## 💬 Support

- **Documentation**: [objectweaver.dev/docs](https://objectweaver.dev/docs)
- **Community Forum**: [objectweaver.dev/community](https://objectweaver.dev/community)
- **Email**: support@objectweaver.dev
- **Enterprise Support**: [Contact Sales](https://objectweaver.dev/enterprise)

---

<div align="center">

**[⭐ Star us on GitHub](https://github.com/objectweaver/objectweaver)** if you find objectweaver useful!

Made with ❤️ by the objectweaver team

</div>