# workflow-plugin-ory-polis

Ory Polis enterprise SSO and Directory Sync provider plugin for Workflow.

Polis does not currently publish a stable official Go SDK client. This plugin
uses a small typed HTTP client over the official Polis OpenAPI surface from
`ory/polis` v26.2.0 instead of advertising a nonexistent SDK-backed client.

## Capabilities

- `ory.polis` module using the Polis API base URL and API key
- Auth provider descriptor step for admin catalog integration
- SSO connection create/read/list/update/delete steps
- SSO and Directory Sync delegated setup-link steps
- Directory Sync connection create/read/list/update/delete steps
- Directory user/group/member/event read steps
- Identity federation read steps

The descriptor advertises only endpoints implemented by this plugin.

## Security

The Polis API key is sent as `Authorization: Bearer <token>`. Store it in a
Workflow secret source, keep Polis management routes server-side, and rotate
the key regularly.

## Install

```sh
wfctl plugin install workflow-plugin-ory-polis
```

## License

MIT
