## Question Handling with oh-my-opencode

### When Sisyphus Asks Questions

Sisyphus (the execution agent) is capable of handling both implementation and minor planning decisions. For most questions, you can **answer directly** without switching to Prometheus.

**Example Sisyphus response:**
```
I'm implementing the user registration feature. Should I require:
- Email verification?
- Phone verification?
- Both?

Also, what password policy should I enforce?
```

### Response Strategy: Direct vs. Switch

**Direct Answer (Recommended for minor decisions):**
- Simple requirements (e.g., validation rules)
- Naming conventions
- Minor implementation details
- Configuration choices

**Switch to Prometheus (For major decisions):**
- Architecture changes
- New feature scope changes
- Breaking changes
- Re-planning needed
- Complex trade-off decisions

### Direct Answer to Sisyphus

**Answer in the same message thread:**
```json
POST /session/:id/message
Body: {
  "agent": "Sisyphus",
  "parts": [{
    "role": "user",
    "content": {
      "type": "text",
      "text": "Require email verification only. Password policy: minimum 8 characters, mixed case, one number."
    }
  }]
}
```

**Why this works:**
- Sisyphus is intelligent and context-aware
- Can handle both planning and execution
- No need to interrupt the flow for simple decisions
- Faster workflow - no agent switching overhead

### When to Switch to Prometheus

**Switch to Prometheus for:**

1. **Architecture decisions**
   ```
   Sisyphus: "Should I use SQL or NoSQL for user data?"
   → Switch to Prometheus (major architectural choice)
   ```

2. **Scope changes**
   ```
   Sisyphus: "This requires adding a new payment gateway."
   → Switch to Prometheus (feature scope expansion)
   ```

3. **Re-planning needed**
   ```
   Sisyphus: "The current approach won't work because..."
   → Switch to Prometheus (need new plan)
   ```

### Switching to Prometheus Workflow

**1. Acknowledge the question:**
   "Sisyphus encountered a major decision point. Let me switch to Prometheus."

**2. Switch to Prometheus:**
   ```json
   POST /session/:id/message
   Body: {
     "agent": "Prometheus",
     "parts": [{
       "role": "user",
       "content": {
         "type": "text",
         "text": "Sisyphus encountered a question: [list question]. Please analyze and help decide the right approach before continuing implementation."
       }
     }]
   }
   ```

**3. Let Prometheus analyze:**
   - Discuss requirements with user
   - Consider pros/cons
   - Propose options
   - Get user decision

**4. Update the plan:**
   - Document the decision
   - Update plan if needed
   - Provide clear guidance

**5. Trigger /start-work:**
   ```json
   POST /session/:id/command
   Body: {
     "command": "/start-work",
     "agent": "Prometheus"
   }
   ```

**6. Sisyphus resumes:**
   Sisyphus automatically takes over after /start-work and continues with the updated plan. No need to send an explicit message.

### Question Types and Handling

**1. Requirements Questions**
> "Should X feature include Y option?"

**Direct Sisyphus response (for minor choices):**
- Answer directly based on requirements
- Keep it simple and consistent
- "Include Y option" or "Don't include Y"

**Prometheus response (for major requirements):**
- Analyze user needs
- Consider project constraints
- Propose recommendation
- Get user confirmation
- Update plan
- Send /start-work

**2. Technical Decisions**
> "Should I use library A or library B?"

**Direct Sisyphus response (if patterns exist):**
- Check existing codebase for similar patterns
- Match existing conventions
- "Use library A (already used in project)"

**Prometheus response (for new technical choices):**
- Compare options
- Evaluate tradeoffs
- Check existing dependencies
- Recommend based on project patterns
- Update plan if needed
- Send /start-work

**3. Edge Cases**
> "What if user input is invalid/malicious?"

**Direct Sisyphus response:**
- Follow standard security practices
- "Validate input and return error for invalid"
- Simple validation logic

**Prometheus response (for complex edge cases):**
- Define validation strategy
- Set error handling approach
- Determine security requirements
- Document edge case handling
- May require plan update

**4. Breaking Changes**
> "This change will affect [other module]. Should I update it too?"

**Always use Prometheus for breaking changes:**
- Assess impact
- Decide on scope
- May need separate plan phase
- Get approval for expanded scope
- Send /start-work with updated plan

### User Cannot Answer

If user doesn't know the answer:

**Sisyphus should (for minor decisions):**
1. Research codebase patterns
2. Check similar implementations
3. Make reasonable recommendation
4. Proceed with default and continue

**Prometheus should (for major decisions):**
1. Research codebase patterns
2. Check similar implementations
3. Make reasonable recommendation
4. Ask for confirmation
5. Document in plan

**Example (Sisyphus - minor):**
```
I don't see any existing password validation. I'll use standard practice:
- Minimum 8 characters
- Mixed case
- One number

Proceeding with this approach.
```

**Example (Prometheus - major):**
```
I don't see any existing authentication in the codebase. I recommend:
- JWT-based authentication
- Email verification for signup
- Standard password policy

Should I proceed with this, or do you have other requirements?
```

### Decision Documentation

After answering questions, **document the decision**:

```
### Decision Log

**Question:** Email verification required?
**Decision:** Yes, require email verification
**Reasoning:** Prevents fake accounts, meets security best practices
**Implications:** Need email service integration, add verification flow

**Question:** Password policy?
**Decision:** Min 8 chars, mixed case + number
**Reasoning:** Standard security practice, balances usability
**Implications:** Update validation logic, add user guidance
```

This documentation helps Prometheus and Sisyphus understand decisions.

### Rapid Question Handling

For multiple questions, handle them in batches:

**Wrong approach:**
```
Question 1 → Prometheus → Answer → /start-work → Sisyphus → Question 2 → Prometheus → Answer → /start-work → ...
```

**Correct approach (minor questions):**
```
Questions 1,2,3 → Answer all to Sisyphus directly → Continue
```

**Correct approach (major questions):**
```
Questions 1,2,3 → Prometheus → Answer all → /start-work → Sisyphus → Continue
```

**Sisyphus asking minor questions:**
```
I need answers to:
1. Email verification required?
2. Phone verification required?
3. Password policy?
```

**Direct answer to Sisyphus:**
```
Let's address all at once:

1. Yes, email verification required
2. No phone verification needed
3. Password: min 8 chars, mixed case, one number

Proceed with these requirements.
```

**No /start-work needed** - just answer directly!

### When to Use /start-work

**Send /start-work when:**
- Prometheus created a new plan
- Prometheus updated an existing plan
- All questions answered
- Requirements documented
- User says "go ahead" or confirms

**Don't send /start-work if:**
- You're just clarifying with Sisyphus (answer directly)
- Plan needs more analysis from Prometheus
- User asks for alternatives
- Minor issues that don't affect the plan

### Example Complete Flow (Minor Questions - Direct to Sisyphus)

**Step 1: Sisyphus encounters minor question**
```
(From Sisyphus)
I'm creating the user model. Should I include:
- Created timestamp?
- Updated timestamp?
- Soft delete (deletedAt)?
```

**Step 2: Check existing patterns**
Review codebase to see what other models use.

**Step 3: Answer directly**
```
(You to Sisyphus)
Use the same pattern as existing models:
- Include createdAt and updatedAt
- Skip deletedAt (not used elsewhere)

Proceed with these timestamp fields.
```

**Step 4: Sisyphus continues**
```
Adding createdAt and updatedAt timestamps to user model...
```

### Example Complete Flow (Major Questions - Switch to Prometheus)

**Step 1: Sisyphus encounters major question**
```
(From Sisyphus)
I need to implement authentication. Should I use:
- JWT tokens?
- Session-based auth?
- OAuth provider?

Also, do we need multi-factor authentication?
```

**Step 2: Switch to Prometheus**
```
(You to Prometheus)
Sisyphus is implementing authentication and needs architectural decisions. Please analyze the requirements and recommend an authentication approach.
```

**Step 3: Prometheus analyzes and recommends**
```
I recommend:
- JWT tokens (stateless, scalable, matches REST API pattern)
- OAuth integration for social login
- No MFA initially (can add later if needed)

Check existing codebase patterns...
[analyzes]

No existing auth found. This will be a new module.
Proposed approach aligns with modern best practices.
```

**Step 4: User confirms**
```
Yes, proceed with JWT + OAuth. No MFA for now.
```

**Step 5: Send /start-work**
```
POST /session/:id/command
Body: {
  "command": "/start-work",
  "agent": "Prometheus"
}
```

**Step 6: Sisyphus automatically resumes**
Sisyphus takes over and begins implementing the authentication system based on Prometheus's plan.

### Special Cases

**1. Sisyphus wants to make major design decisions**

Switch to Prometheus:
```
Sisyphus is attempting to make a major design decision about [X]. Let me switch to Prometheus to analyze this properly.
```

**2. Questions are minor/trivial**

Answer directly to Sisyphus (no switch needed):
- Check existing patterns first
- Answer directly and concisely
- Let Sisyphus continue implementation

**3. Multiple unrelated questions**

Group by importance:
- Minor implementation questions → answer directly to Sisyphus
- Major architectural questions → switch to Prometheus

### Anti-Patterns

**❌ Wrong: Switching to Prometheus for minor questions**
```
(You to Prometheus)
Sisyphus asked about password validation. Let me switch to analyze.
```
Problem: Overhead and unnecessary. Sisyphus can handle minor decisions.

**✅ Correct: Answer directly to Sisyphus**
```
(You to Sisyphus)
Password: minimum 8 characters, mixed case, one number. Proceed.
```

**❌ Wrong: Forgetting /start-work**
```
(You to Prometheus)
Here's my feedback on the plan...
[waiting for Prometheus to continue]
```
Problem: Prometheus won't implement. Need to send `/start-work`.

**✅ Correct: Always /start-work after plan approval**
```
(You to Prometheus)
Feedback looks good. Please update the plan.
[Prometheus updates plan]
POST /session/:id/command
Body: { "command": "/start-work", "agent": "Prometheus" }
```

**❌ Wrong: Switching to Atlas or user-only agents**
```
(You to Atlas)
Switching to Atlas mode.
```
Problem: Atlas is user-only. Won't work for automation.

**✅ Correct: Use Prometheus or Sisyphus**
```
(You to Prometheus)
Let me switch to Prometheus to analyze this decision.
```

### Best Practices

1. **Answer minor questions directly** to Sisyphus (no switch needed)
2. **Switch to Prometheus** only for major architectural decisions
3. **Always send /start-work** after Prometheus creates/updates plan
4. **Document decisions** for future reference
5. **Batch questions** when possible (group by importance)
6. **Check existing patterns** before deciding
7. **Get user confirmation** for important decisions
8. **Never use Atlas or user-only agents** in automated workflows
9. **Monitor message flow** to know which agent is active
10. **Preserve context** - same session automatically preserves it
