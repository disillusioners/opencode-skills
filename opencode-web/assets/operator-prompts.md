### Ask for model
Which AI model would you like to use with oh-my-opencode?

### Request plan via web API (Prometheus)
Send a plan request to Prometheus agent to analyze the task and propose a step-by-step plan. Ask clarification questions if needed.

API call:
```
POST /session/:id/message
Body: {
  "agent": "Prometheus",
  "model": "provider:model",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Analyze the task and propose a step-by-step plan. Ask clarification questions if needed."
    }
  }]
}
```

### Request plan revision via web API (Prometheus)
The plan has issues. Send a revision request to Prometheus agent via OpenCode web API.

API call:
```
POST /session/:id/message
Body: {
  "agent": "Prometheus",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "The plan has issues. Please revise it: [specific issues and concerns]"
    }
  }]
}
```

### Trigger execution: /start-work command
Plan is approved. Send /start-work command to Prometheus to trigger handoff to Sisyphus for implementation.

API call:
```
POST /session/:id/command
Body: {
  "command": "/start-work",
  "agent": "Prometheus"
}
```

This triggers Prometheus to hand off the plan to Sisyphus automatically.

### Request implementation via web API (Sisyphus directly)
For simple tasks, use Sisyphus directly without explicit Prometheus planning.

API call:
```
POST /session/:id/message
Body: {
  "agent": "Sisyphus",
  "model": "provider:model",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Add [feature] following existing patterns in the codebase."
    }
  }]
}
```

### Handle minor question from Sisyphus
Sisyphus encountered minor question. Answer directly without switching to Prometheus.

API call:
```
POST /session/:id/message
Body: {
  "agent": "Sisyphus",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "For the password policy: minimum 8 characters, mixed case, one number. Proceed with this."
    }
  }]
}
```

### Handle major question from Sisyphus
Sisyphus encountered a major architectural decision. Switch to Prometheus to analyze properly.

API call:
```
POST /session/:id/message
Body: {
  "agent": "Prometheus",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Sisyphus encountered a major decision: [decision]. Please analyze and recommend the right approach before continuing implementation."
    }
  }]
}
```

### After Prometheus decision: /start-work
Prometheus has decided on requirements and updated the plan. Send /start-work to trigger Sisyphus implementation.

API call:
```
POST /session/:id/command
Body: {
  "command": "/start-work",
  "agent": "Prometheus"
}
```

Sisyphus will automatically resume with the updated plan.
POST /session/:id/message
Body: {
  "agent": "plan",
  "model": "provider:model",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Analyze the task and propose a step-by-step plan. Ask clarification questions if needed."
    }
  }]
}
```

### Request plan revision via web API
The plan has issues. Send a revision request to the plan agent via OpenCode web API.

API call:
```
POST /session/:id/message
Body: {
  "agent": "plan",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "The plan has issues. Please revise it: [specific issues and concerns]"
    }
  }]
}
```

### Request implementation via web API
Switch to build agent and request implementation based on the approved plan via OpenCode web API.

API call:
```
POST /session/:id/message
Body: {
  "agent": "build",
  "model": "provider:model",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Proceed with implementation based on the approved plan."
    }
  }]
}
```

### Handle question from build agent
Build agent encountered questions. Switch to plan agent to address them before continuing implementation.

API call:
```
POST /session/:id/message
Body: {
  "agent": "plan",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Build agent encountered these questions: [list questions]. Please help decide on these requirements before proceeding with implementation."
    }
  }]
}
```

### Confirm decision and return to build
Plan agent has decided on the requirements. Switch back to build agent to continue implementation.

API call:
```
POST /session/:id/message
Body: {
  "agent": "build",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Plan agent decided: [decisions]. Continue implementation with these requirements."
    }
  }]
}
```

### Check session status
Check the current status of all sessions via OpenCode web API.

API call:
```
GET /session/status
```

### Get session messages
Retrieve recent messages from a session to understand progress.

API call:
```
GET /session/:id/message?limit=10
```

### Abort stuck session
A session is stuck or taking too long. Abort it via OpenCode web API.

API call:
```
POST /session/:id/abort
```

### List available models
Get list of available AI models via OpenCode web API.

API call:
```
GET /config/models
```

### Get session diff
Review the file changes made in the session.

API call:
```
GET /session/:id/diff
```

### Verify server health
Check if OpenCode server is running and accessible.

API call:
```
GET http://127.0.0.1:4096/global/health
```

### List sessions
Get all sessions to find or reuse an existing session.

API call:
```
GET /session
```

### Create new session
Create a new session in OpenCode via web API.

API call:
```
POST /session
Body: { "title": "Session Title" }
```

### List available agents
Get list of available agents (Prometheus, Sisyphus, Hephaestus, etc.) via oh-my-opencode web API.

API call:
```
GET /agent
```
