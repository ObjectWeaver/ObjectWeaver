# ObjectWeaver Enterprise Edition

This directory contains the Enterprise Edition features of ObjectWeaver.

## License

All code in this directory is licensed under the ObjectWeaver Commercial License.
See [LICENSE](LICENSE) for complete terms.

## Enterprise Features

Enterprise Edition includes additional features beyond the Community Edition:

- Advanced monitoring and observability
- Single Sign-On (SSO) integration
- Multi-tenancy support
- Advanced security features
- Priority support
- Custom integrations
- Enhanced SLAs

## Usage

To use Enterprise Edition features, you need a valid ObjectWeaver Enterprise Edition subscription.

For inquiries, visit: https://objectweaver.dev/contact

## Development

When adding new enterprise features:

1. Place all code in this `ee/` directory
2. Add the appropriate license header to each file (see `LICENSE_HEADERS.md` in the root)
3. Ensure the feature gracefully degrades or is disabled if no valid license is present
4. Document the feature in the Enterprise Edition documentation

## Structure

```
ee/
├── LICENSE                    # ObjectWeaver Commercial License
├── README.md                  # This file
├── auth/                      # Advanced authentication features
├── monitoring/                # Enhanced monitoring capabilities
├── multitenancy/              # Multi-tenant features
└── ... (other enterprise features)
```
