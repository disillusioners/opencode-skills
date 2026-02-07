## Standard oh-my-opencode Web Workflow

### 1. Pre-flight Check
```
GET /global/health
```
Verify OpenCode server is running and accessible.

### 2. Session Setup
```
GET /session
```
List all sessions. Check if one exists for current project.

**If existing session found:**
- Record the session ID
- Get session details: `GET /session/:id`
- Review recent messages: `GET /session/:id/message?limit=5`

**If no existing session:**
- Ask user for approval
- Create new session: `POST /session` with title

### 3. Select Model
```
GET /config/providers
```
List available models from providers and select one.

- Use the default model if user doesn't specify
- Use the model ID in message requests

### 4. List Available Agents
```
GET /agent
```
Get list of agents and their capabilities.
- **Prometheus**: For analysis, planning, and architecture
- **Sisyphus**: For implementation and general coding (default agent)
- **Hephaestus**: For automation tasks
- Atlas and others: USER-ONLY - do not use for automation

### 5. Planning Phase (for complex tasks)

**Send plan request to Prometheus:**
```
POST /session/:id/message
Body: {
  "agent": "Prometheus",
  "model": "provider_id/model_id",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Analyze the task and propose a step-by-step plan. Ask clarification questions if needed."
    }
  }]
}
```

**Review response:**
- Check if plan is complete and accurate
- Look for clarification questions
- If questions exist, answer them in follow-up messages

**Revise if needed:**
```
POST /session/:id/message
Body: {
  "agent": "Prometheus",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "The plan has issues. Please revise it: [specific issues]"
    }
  }]
}
```

### 6. Trigger Execution with /start-work

**CRITICAL**: Send /start-work command when plan is approved:
```
POST /session/:id/command
Body: {
  "command": "/start-work",
  "agent": "Prometheus"
}
```

This triggers Prometheus to hand off the plan to Sisyphus for implementation.

**What happens:**
- Prometheus creates work items (todo list)
- Hands off to Sisyphus automatically
- Sisyphus begins implementing the plan

### 7. Implementation Phase (Sisyphus)

**Monitor Sisyphus execution:**
```
GET /session/:id/message?limit=20
```
Check for Sisyphus messages implementing the plan.

**Check session status:**
```
GET /session/status
```
Verify session is running or completed.

**Check todo list:**
```
GET /session/:id/todo
```
Track which work items are completed.

### 8. Handle Clarifications

If Sisyphus asks questions:
1. **Answer directly** in the same message thread (no agent switch needed)
2. Sisyphus can handle both planning and execution context
3. Continue monitoring progress

For major issues requiring re-planning:
1. **Switch to Prometheus** to discuss the issue
2. **Send /start-work** again after new plan is approved
3. **Sisyphus resumes** with updated plan

### 9. Review and Iterate

**Check session diff:**
```
GET /session/:id/diff
```
Review all file changes made.

**Get todo list:**
```
GET /session/:id/todo
```
Check what items are completed.

**Repeat Prometheus → /start-work → Sisyphus loop** until:
- All requirements satisfied
- User confirms completion
- No more work needed

### 9. Finalization

- Review final session state
- Confirm all files changed as expected
- Document any manual steps required
- Mark task as complete

### Error Recovery

| Issue | Recovery Action |
|-------|-----------------|
| Session stuck | `POST /session/:id/abort` |
| Wrong model used | Send new message with correct model |
| Prometheus not implementing | Send `/start-work` command |
| Agent error | Switch to Sisyphus (default) and retry |
| Timeout | Use `/prompt_async` endpoint |

### Async Workflow for Long Tasks

For long-running operations:
```
POST /session/:id/prompt_async
Body: { ...same as message... }
```

Monitor via:
- `GET /session/:id/message` - poll for new messages
- `GET /session/status` - check if still running
- `GET /session/:id/todo` - track todo progress
