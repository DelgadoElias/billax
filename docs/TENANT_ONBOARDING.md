# Tenant Onboarding Guide

This guide walks tenants (SaaS customers) through the process of creating accounts in billax and managing their API keys securely.

---

## Prerequisites

- A billax deployment URL (e.g., https://billax.example.com)
- An email address for account recovery
- A secure password manager or vault for storing API keys

---

## 1. Sign up for an account

Every new tenant starts with `POST /v1/signup`, a public endpoint (no auth required).

```bash
curl -s -X POST https://billax.example.com/v1/signup \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ACME Corp",
    "email": "admin@acme.com"
  }'
```

Response (HTTP 201):
```json
{
  "tenant": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "ACME Corp",
    "slug": "acme-corp",
    "email": "admin@acme.com",
    "is_active": true,
    "created_at": "2026-04-11T12:36:41Z"
  },
  "api_key": {
    "id": "74811871-6350-4c82-aefb-916a495d89c2",
    "key": "payd_live_pc6uiKhGIKgLT8oflp8gheH8RPyF9i2B0-HDZA9MbaI=",
    "key_prefix": "payd_live_pc",
    "scopes": ["read", "write"],
    "created_at": "2026-04-11T12:36:41Z"
  },
  "warning": "Store this key securely. It will not be shown again."
}
```

**⚠️ Important:**
- The `api_key.key` field is shown **once** during signup. Store it immediately in a secure password manager.
- If lost, you must revoke it and create a new key via the authenticated `/v1/keys` endpoint.
- The `slug` is auto-generated from your company name (lowercase, spaces → hyphens).

---

## 2. API Key Management

API keys are the credentials tenants use to call billax endpoints.

### Create a new API key

Once you have your initial key, you can create additional keys via `POST /v1/keys`. This is useful for:
- **Rotating keys** every 6 months (security best practice)
- **Per-service keys** (separate keys for CI/CD, backend service, mobile app)
- **Temporary keys** for contractor access with planned revocation

```bash
# Use your existing API key to create a new one
curl -s -X POST https://billax.example.com/v1/keys \
  -H "Authorization: Bearer $INITIAL_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI/CD - GitHub Actions"
  }'
```

Response (HTTP 201):
```json
{
  "id": "new-key-id-uuid",
  "key": "payd_live_newkeynewkeynewkey=",
  "key_prefix": "payd_live_new",
  "scopes": ["read", "write"],
  "created_at": "2026-04-11T13:00:00Z"
}
```

Store this key immediately in your CI/CD secret manager (GitHub Secrets, GitLab Variables, etc.).

### List your API keys

Retrieve all active keys:

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  https://billax.example.com/v1/keys
```

Response (HTTP 200):
```json
{
  "keys": [
    {
      "id": "74811871-6350-4c82-aefb-916a495d89c2",
      "key_prefix": "payd_live_pc",
      "scopes": ["read", "write"],
      "created_at": "2026-04-11T12:36:41Z"
    },
    {
      "id": "new-key-id-uuid",
      "key_prefix": "payd_live_new",
      "scopes": ["read", "write"],
      "created_at": "2026-04-11T13:00:00Z"
    }
  ]
}
```

**Note:** Plaintext keys are NEVER shown again. Only the prefix (first 12 chars) is displayed for identification.

### Revoke an API key

When rotating keys or revoking contractor access:

```bash
curl -s -X DELETE https://billax.example.com/v1/keys/{keyID} \
  -H "Authorization: Bearer $ACTIVE_API_KEY"
```

Response (HTTP 204): No content — key is revoked immediately.

All requests using the revoked key will fail with `401 Unauthorized`.

---

## 3. Best Practices for API Key Security

### ✅ DO

- **Store keys in a password manager** (1Password, LastPass, Bitwarden, etc.)
- **Use separate keys per environment** (test keys for development, live keys for production)
- **Use separate keys per service** (one for your web backend, one for billing CLI, one for CI/CD)
- **Rotate keys every 6 months** by creating a new key, updating your app, then revoking the old one
- **Revoke keys immediately** if you suspect they were exposed
- **Document where each key is used** (e.g., "payd_live_abc is used in production API server")
- **Restrict key scopes** at the server level (if your billax deployment supports per-key scopes)

### ❌ DON'T

- **Commit API keys to version control** — use `.env` files (local only) or secrets managers
- **Log or print API keys** anywhere (stdout, error logs, crash reports)
- **Hardcode keys in source code** — use environment variables
- **Share keys via email or chat** — use a secure secrets manager
- **Use the same key across multiple services** — rotate is harder; one leak affects everything
- **Forget keys in test scripts** — always use dummy keys or environment variables

---

## 4. Key Naming Convention

When you create a key, include a meaningful name to track its purpose:

```
"CI/CD - GitHub Actions (production)"
"Mobile backend - us-west-2"
"Development - MacBook Pro"
"Stripe data sync service"
"Contractor ABC - temporary (expires 2026-06-01)"
```

This helps you quickly identify which key to revoke if needed.

---

## 5. What to do if you lose an API key

**If a key is lost or potentially exposed:**

1. **Revoke the key immediately:**
   ```bash
   curl -s -X DELETE https://billax.example.com/v1/keys/{keyID} \
     -H "Authorization: Bearer $ANOTHER_ACTIVE_KEY"
   ```

2. **Create a new key:**
   ```bash
   curl -s -X POST https://billax.example.com/v1/keys \
     -H "Authorization: Bearer $ANOTHER_ACTIVE_KEY" \
     -H "Content-Type: application/json" \
     -d '{"name": "Replacement after security incident"}'
   ```

3. **Update your application** with the new key (environment variables, config, secrets manager)

4. **Monitor activity** for the old key in your logs — if you see requests, contact support

**Why multiple keys matter:** If you always have at least 2 active keys, losing one key doesn't block your service while you create a replacement.

---

## 6. Getting your account info

Retrieve your tenant information (name, slug, email):

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  https://billax.example.com/v1/me
```

Response (HTTP 200):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "ACME Corp",
  "slug": "acme-corp",
  "email": "admin@acme.com",
  "is_active": true,
  "created_at": "2026-04-11T12:36:41Z"
}
```

---

## 7. API Key Scopes (future)

Currently, all keys have scopes `["read", "write"]` (full access). Future versions may support fine-grained scopes:

- `read:plans` — list and retrieve plans
- `write:plans` — create and modify plans
- `read:subscriptions` — list and retrieve subscriptions
- `write:subscriptions` — create, modify, and cancel subscriptions
- `read:payments` — list and retrieve payments
- `write:payments` — create payments

When this is released, you can create keys with minimal required permissions. For now, treat all keys as full-access credentials.

---

## 8. Troubleshooting

**"API key not found" (401 Unauthorized)**
- Verify the API key is correct (copy-paste, no extra spaces)
- Check that the key hasn't expired or been revoked
- Confirm you're using the correct API key format (`payd_live_...` or `payd_test_...`)

**"Cannot create a second key" (500 error)**
- This is a bug — contact support with your tenant ID
- As a workaround, use your initial signup key for now

**"I lost my initial key and can't create a new one"**
- You'll need to contact the billax admin to reset your account or issue a temporary key
- This is why having 2+ active keys is important

**"How do I know which key is which?"**
- The `key_prefix` (first 12 chars) is displayed in list responses — match it against your stored keys
- Document the purpose of each key in your password manager

---

## Next Steps

1. **Set up your first plan** — see [GETTING_STARTED.md](GETTING_STARTED.md) step 3
2. **Create subscriptions** — see [GETTING_STARTED.md](GETTING_STARTED.md) step 4
3. **Integrate webhooks** — see [OPERATIONS.md](OPERATIONS.md) for production setup
4. **Monitor your usage** — visit your billax dashboard (if available) or use the `/v1/me` endpoint to confirm your account

---

## FAQ

**Q: Can I change my API key?**
A: No, but you can revoke the current key and create a new one. The new key is independent.

**Q: What if my API key is leaked on GitHub?**
A: Revoke it immediately. GitHub will alert you if they detect exposed keys. Create a new key and force-push your repo (removing the exposed key from history).

**Q: Do API keys have an expiration date?**
A: Not by default, but you should rotate every 6 months as a security best practice.

**Q: Can I use the same key across multiple services?**
A: Technically yes, but it's not recommended. If one service is compromised, all services are affected. Use separate keys.

**Q: How many keys can I create?**
A: No limit, but create keys purposefully (per service or per rotation cycle). Hundreds of unused keys clutter your account.

