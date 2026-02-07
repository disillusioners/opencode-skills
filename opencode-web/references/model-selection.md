## Model Selection

### List Models
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
      "models": {
        "gpt-4": { "id": "gpt-4", "name": "GPT-4" },
        "gpt-3.5-turbo": { ... }
      }
    },
    {
      "id": "anthropic",
      "name": "Anthropic",
      "models": {
        "claude-3-opus": { "id": "claude-3-opus", "name": "Claude 3 Opus" }
      }
    }
  ],
  "default": {
    "openai": "gpt-4",
    "anthropic": "claude-3-opus"
  }
}
```

**Extracting Models:**
1. Iterate through the `providers` array.
2. For each provider, access its `models` object.
3. Construct the full model ID using the format `providerID:modelID` (e.g., `openai:gpt-4`).

### Model Selection Strategy

1. **Ask user preference first:**
   "Which AI model would you like to use?"

2. **If user specifies model:**
   - Check if the model exists in the `providers` list
   - Use the format `providerID:modelID`

3. **If user doesn't specify:**
   - Use the `default` mapping from `/config/providers` to find the default model for a preferred provider.
   - Or ask the user to clarify.

4. **Model ID format:**
   - ALWAYS use `providerID:modelID`
   - Examples: `openai:gpt-4`, `anthropic:claude-3-opus`

### Applying Model in Messages

**Specify in message body:**
```json
{
  "model": "openai:gpt-4",
  "agent": "plan",
  "parts": [...]
}
```

**Omit model field:**
- OpenCode will use the session's default model or the last used model.

### Troubleshooting

| Issue | Solution |
|-------|----------|
| Model not found | Verify model ID format `providerID:modelID` matches available models from `/config/providers` |
| Server error | Ensure `opencode serve` is running and accessible `GET /global/health` |

### Best Practices

1. **Use default models** unless user specifies otherwise
2. **Ask user** if multiple suitable models are available
3. **Cache model list** to avoid repeated calls to `/config/providers`
4. **Validate model ID** before sending message requests
