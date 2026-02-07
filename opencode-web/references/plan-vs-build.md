## Prometheus (Plan) vs Sisyphus (Execute)

### Prometheus Agent (Planning)

**Purpose:**
- Analyze tasks and understand requirements
- Create detailed step-by-step implementation plans
- Ask clarification questions
- Review and revise plans
- Architecture and design decisions
- **Special**: Requires `/start-work` command to trigger execution

**When to use:**
- Complex tasks requiring detailed planning
- Architecture decisions needed
- New features or major refactors
- User asks "how should I..."
- Need to understand existing codebase before changes
- Before any significant code changes

**Prometheus agent rules:**
- Focus on analysis and planning
- Ask questions to clarify requirements
- Review existing code and patterns
- Identify dependencies and risks
- Propose multiple approaches if applicable
- Minimal code examples only for clarity (not full implementation)

**Example plan request:**
```json
{
  "agent": "Prometheus",
  "model": "openai:gpt-4",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "I need to add user authentication to my API. Analyze the task and propose a step-by-step plan. Consider security, existing patterns, and dependencies."
    }
  }]
}
```

**Plan review checklist:**
- [ ] Requirements clearly understood
- [ ] Dependencies identified
- [ ] Steps are logical and complete
- [ ] Security concerns addressed
- [ ] Testing strategy included
- [ ] No unnecessary code generation

### Triggering Execution: /start-work

**CRITICAL**: Prometheus does NOT implement code. You must send `/start-work` to trigger execution.

**When plan is approved:**
```json
POST /session/:id/command
Body: {
  "command": "/start-work",
  "agent": "Prometheus"
}
```

**What /start-work does:**
- Prometheus hands off the approved plan to Sisyphus
- Sisyphus begins implementation based on the plan
- Context is preserved (same session)
- No need to switch agents manually

**After /start-work:**
- Monitor messages for Sisyphus activity
- Answer any questions Sisyphus asks directly
- Sisyphus implements the plan step by step

### Sisyphus Agent (Execution)

**Purpose:**
- Implement approved plans (from Prometheus)
- Write and modify code
- Run tests and verify functionality
- Fix bugs and handle edge cases
- File operations and refactoring
- Can also handle simple planning on its own

**When to use:**
- After /start-work triggers execution
- For simple tasks that don't need explicit Prometheus planning
- Direct coding tasks with clear requirements
- Running tests or builds
- Refactoring existing code
- Fixing bugs

**Sisyphus agent rules:**
- Implement based on approved plan or user request
- Use existing patterns and conventions
- Run tests to verify changes
- Can ask questions - answer them directly in the same thread
- Capable of both planning and execution as needed

**Example direct request (no Prometheus needed):**
```json
{
  "agent": "Sisyphus",
  "model": "openai:gpt-4",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Add a user registration form with email and password fields. Follow the existing form patterns in the project."
    }
  }]
}
```

**After /start-work execution:**
```json
{
  "agent": "Sisyphus",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Continue with the next step in the authentication implementation."
    }
  }]
}
```

**Build verification:**
- [ ] Tests pass (if applicable)
- [ ] Code follows existing patterns
- [ ] No type errors or lint issues
- [ ] Files changed as expected
- [ ] Documentation updated if needed

### Hephaestus Agent (Automation)

**Purpose:**
- Specialized automation tasks
- Workflow automation
- Repetitive or scripted operations
- Batch operations

**When to use:**
- Automation-specific tasks
- Scripted workflows
- Multi-step automation sequences
- User explicitly requests Hephaestus

**Example automation request:**
```json
{
  "agent": "Hephaestus",
  "model": "openai:gpt-4",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Automate the deployment process for staging environment."
    }
  }]
}
```

### Forbidden Agents

**Atlas and other agents:**
- These are USER-ONLY agents
- Designed for TUI interaction
- Do NOT use in automated workflows
- They will not respond correctly to programmatic control

**Always use:**
- Prometheus for planning
- Sisyphus for execution (default)
- Hephaestus for automation

### Workflow Patterns

**Standard Prometheus → Sisyphus loop:**
```
Prometheus (plan) → Review → /start-work → Sisyphus (execute) → Verify
```

**Example:**
1. **Prometheus**: "Add authentication to API"
   - Prometheus: analyzes, creates 5-step plan
2. **Review**: User reviews plan, asks clarifying question
   - Prometheus: answers, refines plan
3. **Approve**: User says "looks good"
4. **/start-work**: Trigger execution
   - Command sent to Prometheus
5. **Sisyphus**: Begins implementation
   - Implements steps 1-2
   - Asks question about configuration
6. **Answer**: User responds to Sisyphus directly
   - Sisyphus: continues with step 3-5
7. **Verify**: User tests, finds issue
8. **Prometheus**: Discuss fix approach
9. **/start-work**: Trigger fix implementation
10. **Sisyphus**: Implements fix
11. **Done**: Task complete

**Direct Sisyphus workflow:**
```
Sisyphus (plan + execute) → Verify → Done
```
Example: "Add a simple button component"

**Complex multi-phase:**
```
Prometheus → Review → /start-work → Sisyphus → Issue → Prometheus → /start-work → Sisyphus → Done
```
Example: "Rewrite authentication system with issues"

### Agent Selection Decision Tree

**Complex task? Architecture needed? Major feature?**
→ Use Prometheus → /start-work → Sisyphus

**Simple task? Clear requirements?**
→ Use Sisyphus directly (default agent)

**Automation task? Workflow automation?**
→ Use Hephaestus

**User asks "how should I..." or "what's the best approach..."**
→ Use Prometheus (planning focus)

**User says "implement this specific thing" with clear specs**
→ Use Sisyphus directly

**User wants to "analyze" or "understand" existing code**
→ Use Prometheus (analysis focus)

### Monitoring Agent Activity

Check which agent is active:
```
GET /session/:id/message?limit=10
```
Look at each message's `agent` field to track transitions.

```
GET /session/status
```
Shows which agent is currently processing.

### Troubleshooting

| Issue | Solution |
|-------|----------|
| Prometheus not executing code | Send `/start-work` command |
| Sisyphus asks for plan | Let Sisyphus plan, or switch to Prometheus |
| /start-work not working | Check it's sent to Prometheus agent |
| Agent not switching | Send new message with `agent` field set |
| Context lost between agents | Same session preserves context automatically |
| Wrong agent behavior | Check agent ID: Prometheus, Sisyphus, Hephaestus |
| Atlas agent issues | Don't use Atlas - it's user-only |

### Best Practices

1. **Use Prometheus** for complex tasks requiring detailed planning
2. **Always /start-work** after Prometheus plan approval
3. **Use Sisyphus directly** for simple, clear tasks
4. **Answer Sisyphus questions** directly (no need to switch)
5. **Review plans** before /start-work
6. **Verify Sisyphus output** matches Prometheus plan
7. **Iterate as needed** - Prometheus / /start-work / Sisyphus as many times as required
8. **Never use Atlas or other user-only agents** in automated workflows
9. **Check agent IDs** - use exact names: "Prometheus", "Sisyphus", "Hephaestus"
10. **Monitor message flow** to track agent transitions

### /start-work Command Details

**Endpoint:**
```
POST /session/:id/command
Body: {
  "command": "/start-work",
  "agent": "Prometheus"
}
```

**When to send:**
- After Prometheus plan is complete
- User has reviewed and approved the plan
- All clarifying questions answered
- Ready to begin implementation

**What happens:**
- Prometheus finalizes the plan
- Creates work items (todo list)
- Hands off to Sisyphus automatically
- Sisyphus starts first work item

**Error cases:**
- If plan is incomplete: Prometheus will ask for more details
- If no approval needed: Send anyway to trigger handoff
- If issues during /start-work: Check Prometheus response for errors
