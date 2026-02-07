## Model Selection and Authentication

### List Providers and Models
```
GET /config/providers
```
Response:
```json
{
  "providers": [
    {
      "id": "openai",
      "name": "OpenAI",
      "models": ["gpt-4", "gpt-3.5-turbo", ...]
    },
    {
      "id": "anthropic",
      "name": "Anthropic",
      "models": ["claude-3-opus", "claude-3-sonnet", ...]
    }
  ],
  "default": {
    "openai": "gpt-4",
    "anthropic": "claude-3-opus"
  }
}
```

### Determine Authentication Requirements
```
GET /provider/auth
```
Response:
```json
{
  "openai": [
    { "type": "apiKey", "fields": ["apiKey"] }
  ],
  "anthropic": [
    { "type": "apiKey", "fields": ["apiKey"] }
  ],
  "google": [
    { "type": "oauth", "scopes": ["..."] }
  ],
  "ollama": []  // No auth required (local)
}
```

### OAuth Flow

**1. Initiate OAuth:**
```
POST /provider/{id}/oauth/authorize
```
Response:
```json
{
  "authorizationUrl": "https://...",
  "state": "random_state_string"
}
```

**2. Send URL to user:**
"Please visit this URL to authorize [Provider]: [authorizationUrl]"

**3. Wait for user confirmation:**
- User completes OAuth flow in browser
- Provider redirects back to OpenCode server
- OpenCode stores the tokens automatically
- User confirms completion

**4. Verify authentication:**
```
GET /provider
```
Check if provider ID appears in `connected` array.

### API Key Flow

**1. Ask user for API key:**
"What is your [Provider] API key?"

**2. Set credentials:**
```
PUT /auth/{providerID}
Body: {
  "apiKey": "sk-..."
}
```
Body structure varies by provider - check their schema.

**3. Verify success:**
Response `true` indicates success.

### Local Providers (No Auth)

For local models like Ollama, no authentication is needed:
```
POST /session/:id/message
Body: {
  "model": "ollama:llama3"  // Just use directly
}
```

### Model Selection Strategy

1. **Ask user preference first:**
   "Which AI provider would you like to use?"

2. **If user specifies provider:**
   - Check if provider is available in `/config/providers`
   - Determine auth method from `/provider/auth`
   - Complete authentication flow
   - Use default model for provider, or ask for specific model

3. **If user doesn't specify:**
   - Use provider with default model from `/config/providers`
   - Prefer authenticated providers over local (if available)
   - Fall back to first available provider

4. **Model ID format:**
   - Always use `providerID:modelID` format
   - Examples: `openai:gpt-4`, `anthropic:claude-3-opus`, `ollama:llama3`

### Applying Model in Messages

**Specify in message body:**
```json
{
  "model": "openai:gpt-4",
  "agent": "plan",
  "parts": [...]
}
```

**Omit to use session default:**
```json
{
  "agent": "build",
  "parts": [...]
}
```
If `model` is omitted, uses the last model used in session or session default.

### Verification Commands

**Check connected providers:**
```
GET /provider
Response: { all: [...], default: {...}, connected: ["openai", "anthropic"] }
```

**Verify specific provider:**
Check if provider ID is in `connected` array.

### Troubleshooting

| Issue | Solution |
|-------|----------|
| Provider not found | Check `/config/providers` for available providers |
| Auth required but not configured | Run auth flow before sending messages |
| OAuth URL not working | Verify server is reachable, try new authorize request |
| API key rejected | Check key format, expiration, and provider requirements |
| Model not found | Verify `providerID:modelID` format matches available models |

### Best Practices

1. **Always check auth requirements** before sending messages
2. **Use default models** unless user specifies otherwise
3. **Verify authentication** with `/provider` endpoint
4. **Ask user** before collecting sensitive credentials
5. **Handle OAuth gracefully** - wait for user confirmation
6. **Cache provider list** to avoid repeated calls
7. **Prefer configured providers** over asking user to setup
