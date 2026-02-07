## Session Management via Web API

### List all sessions
```
GET /session
```
Returns array of sessions with:
- `id`: Unique session identifier
- `title`: Session title
- `createdAt`: Timestamp
- `updatedAt`: Timestamp
- `projectID`: Associated project

### Create a new session
```
POST /session
Body: { parentID?, title? }
```
- `parentID`: Optional, fork from existing session
- `title`: Optional session title

### Get specific session
```
GET /session/:id
```
Returns full session details including metadata and configuration.

### Update session
```
PATCH /session/:id
Body: { title? }
```
Update the session title.

### Delete session
```
DELETE /session/:id
```
Deletes the session and all its data. Cannot be undone.

### Session selection workflow

1. **List sessions** to find existing ones
   ```
   GET /session
   ```

2. **Check for existing project session**
   - Compare session titles with project name
   - Check if project path matches current project
   - If match found: reuse that session

3. **Create new session** only if:
   - No existing session for current project
   - User explicitly requests new session

4. **Reuse session** to preserve:
   - Conversation history
   - Previous decisions and context
   - Todo list
   - File changes

### Get session status
```
GET /session/status
```
Returns status object with session ID keys:
- Running/inactive state
- Last message timestamp
- Current activity

### Get session todo list
```
GET /session/:id/todo
```
Returns array of todo items with status and priority.

### Abort running session
```
POST /session/:id/abort
```
Stops any in-progress AI operations. Use when:
- Session is stuck
- Need to cancel long-running task
- Agent is misbehaving

### Get session diff
```
GET /session/:id/diff?messageID=
```
Returns file changes made in the session:
- `path`: File path
- `changes`: Additions, deletions, modifications
- Can filter by specific messageID

### Best practices

1. **Always reuse sessions** for same project
2. **Check before creating** - list sessions first
3. **Ask user** before creating new session for existing project
4. **Use descriptive titles** for easy identification
5. **Check session status** before sending messages
6. **Review diff** to understand what changed
