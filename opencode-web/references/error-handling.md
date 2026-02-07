## Error Handling and Troubleshooting

### Server Connection Errors

**Server not reachable**
```
Error: Connection refused / ECONNREFUSED
```

**Diagnosis:**
```
GET /global/health
```

**Solutions:**
1. Ask user to start OpenCode server: `opencode serve`
2. Verify correct hostname/port (default: 127.0.0.1:4096)
3. Check firewall settings
4. Confirm server is running: `opencode serve` should be active

**Example response:**
"OpenCode server is not running. Please start it with:
```bash
opencode serve
```
Then I'll try again."



### Session Errors

**404 Session not found**

**Diagnosis:**
```
GET /session
```
List all sessions to verify session ID.

**Solutions:**
1. Session might have been deleted
2. Wrong session ID (typo)
3. Session not created yet
4. Create new session if needed

**Example response:**
"Session [id] not found. Available sessions: [list]. Should I create a new session or use an existing one?"

**400 Invalid session state**

**Diagnosis:**
```
GET /session/:id
```
Check session status and metadata.

**Solutions:**
1. Session might be corrupted
2. Try creating new session
3. Contact OpenCode support if persistent

**409 Conflict (Session locked)**

**Diagnosis:**
```
GET /session/status
```
Check if session is in use.

**Solutions:**
1. Wait for current operation to complete
2. Use POST /session/:id/abort to stop
3. Create a new session if needed

### Agent Errors

**400 Invalid agent ID**

**Diagnosis:**
```
GET /agent
```
List all available agents.

**Solutions:**
1. Agent ID might be misspelled
2. Use correct agent ID from list
3. Common IDs: `plan`, `build`

**Example response:**
"Agent [id] not found. Available agents: [list]. Using default: [first agent]"

### Model Errors

**400 Invalid model ID**

**Diagnosis:**
```
GET /config/providers
```
List available models from providers.

**Solutions:**
1. Use correct model ID (`providerID:modelID`)
2. Verify provider and model exist in list

**Example response:**
"Model [id] not found. Available models: [list]. Which model would you like to use?"

### Message Errors

**400 Invalid message format**

**Diagnosis:**
Check request body format against API spec.

**Common issues:**
- Missing `parts` array
- Invalid part structure
- Wrong role values (must be "user", "system", or "assistant")
- Malformed content object

**Example fix:**
```json
{
  "agent": "plan",
  "parts": [{  // Required
    "role": "user",  // Valid: user|system|assistant
    "content": {
      "type": "text",
      "text": "Your message"
    }
  }]
}
```

**408 Request Timeout**

**Diagnosis:**
Message took too long to complete.

**Solutions:**
1. Use `POST /session/:id/prompt_async` for async
2. Poll for results via `GET /session/:id/message`
3. Check if session is stuck via `GET /session/status`
4. Abort if needed: `POST /session/:id/abort`

**500 Internal server error**

**Diagnosis:**
OpenCode server encountered an error.

**Solutions:**
1. Check server logs for details
2. Try the request again
3. Restart server if persistent
4. Check OpenCode documentation for known issues

### File Operation Errors

**404 File not found**

**Diagnosis:**
Verify file path is correct.

**Solutions:**
1. List directory: `GET /file?path=/`
2. Search for file: `GET /find/file?query=filename`
3. Use correct relative path from project root

**403 Permission denied**

**Diagnosis:**
Check file permissions.

**Solutions:**
1. Verify you have read/write access
2. Check file is not locked
3. Try as user with proper permissions

### Session Stuck / Frozen

**Symptoms:**
- GET /session/status shows "running" indefinitely
- No new messages appearing
- Agent not responding

**Solutions:**

**1. Check session status:**
```
GET /session/:id
```
Look at `status` field.

**2. Abort if stuck:**
```
POST /session/:id/abort
```

**3. Start fresh:**
- Create new session
- Copy context from old session if needed

**4. Restart server:**
As user to restart `opencode serve`

### Network Errors

**ECONNRESET / ETIMEDOUT**

**Diagnosis:**
Network connectivity issue.

**Solutions:**
1. Check internet connection
2. Verify server is running
3. Try again after brief delay
4. Check for firewall/proxy issues

### Rate Limiting

**429 Too Many Requests**

**Diagnosis:**
Sending requests too quickly.

**Solutions:**
1. Slow down request rate
2. Implement exponential backoff
3. Check provider rate limits
4. Use async endpoints for long tasks

### Diagnostic Commands

**Full health check:**
```bash
# Server health
GET /global/health

# Config info
GET /config/providers

# Sessions list
GET /session

# Session status
GET /session/status

# Current project
GET /project/current
```

### Error Recovery Flow

**General pattern:**
1. **Identify error type** from HTTP status code or message
2. **Run diagnostic** command(s) for context
3. **Determine cause** (misconfiguration, network, etc.)
4. **Propose solution** to user
5. **Verify fix** by retrying original request
6. **Document issue** for future reference

### Prevention Strategies

1. **Always check health** before operations
2. **Validate input** before sending (model IDs, session IDs)
3. **Check session status** before new messages
5. **Use async** for long-running operations
6. **Implement timeout** handling
7. **Log errors** for debugging

### When to Escalate

Escalate to user/support if:
- Persistent 500 errors from server
- Unknown error types not documented
- Corrupted sessions that won't recover
- Repeated network issues despite server running

### Example Error Messages to User

**Friendly, actionable messages:**

```
❌ Connection failed
OpenCode server is not reachable at http://127.0.0.1:4096

To fix:
1. Start server: opencode serve
2. Verify port is correct
3. Check for firewall blocking

I'll retry once you've confirmed the server is running.
```



```
🔄 Session stuck
The session is taking longer than expected.

Options:
1. Wait a bit longer (recommended for first attempt)
2. Abort and retry
3. Start a new session

What would you like to do?
```
