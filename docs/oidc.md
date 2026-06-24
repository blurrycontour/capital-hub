# OIDC / SSO Setup

Capital Hub supports single sign-on via any OpenID Connect (OIDC) provider
(Authelia, Keycloak, Authentik, etc.).

## Redirect URI

Register the following redirect URI in your identity provider:

```
https://<your-domain>/api/v1/auth/oidc/callback
```

> **Common mistake:** the path is `/api/v1/auth/oidc/callback`, not `/callback`
> or `/oidc/callback`.

## Configuration

OIDC can be configured in two ways — environment variables (recommended for
server deployments) or the admin UI (Settings → OIDC / SSO). Environment
variables always take priority over UI values.

### Environment variables

| Variable                    | Default | Description                                                    |
| --------------------------- | ------- | -------------------------------------------------------------- |
| `CH_OIDC_ENABLED`           | `false` | Set to `true` to enable OIDC login                            |
| `CH_OIDC_ISSUER_URL`        | —       | Provider base URL (e.g. `https://auth.example.com`)           |
| `CH_OIDC_CLIENT_ID`         | —       | Client ID registered with the provider                        |
| `CH_OIDC_CLIENT_SECRET`     | —       | Client secret                                                  |
| `CH_OIDC_REDIRECT_URL`      | —       | Full redirect URI: `https://<host>/api/v1/auth/oidc/callback` |
| `CH_OIDC_ADMIN_GROUP`       | —       | Group claim value that grants the Administrator role          |
| `CH_OIDC_PROVIDER_NAME`     | `OIDC`  | Display name shown on the login button                        |
| `CH_OIDC_ALLOW_REGISTRATION`| `true`  | Set to `false` to prevent new accounts being created via OIDC |

### Admin UI

Navigate to **Admin → Settings** and scroll to the **OIDC / SSO** section.
Fields configured via environment variables are marked **env** and are
read-only in the UI.

## Authelia example

```yaml
identity_providers:
  oidc:
    clients:
      - client_name: 'Capital Hub'
        client_id: 'capital-hub'
        client_secret: '$argon2id$...'    # hashed secret
        public: false
        authorization_policy: 'one_factor'
        redirect_uris:
          - 'https://app.example.com/api/v1/auth/oidc/callback'
        scopes:
          - 'openid'
          - 'email'
          - 'profile'
          - 'groups'
        response_types:
          - 'code'
        grant_types:
          - 'authorization_code'
        token_endpoint_auth_method: 'client_secret_basic'
```

Set `CH_OIDC_ADMIN_GROUP` to the group name (e.g. `admins`) whose members
should automatically receive the Administrator role.

## Account linking

If a user already has a local email/password account, logging in via OIDC with
the **same email address** will automatically link the OIDC identity to the
existing account — no duplicate account is created.

## Disabling registration

To prevent unknown users from creating accounts via OIDC (allow only
pre-existing users to log in with SSO):

```env
CH_OIDC_ALLOW_REGISTRATION=false
```

Users who already have a local account with a matching email can still log in.
