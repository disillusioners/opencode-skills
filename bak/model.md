### List models
```
GET /config/providers
Response: { providers: Provider[], default: { ... } }
```
Extract models from the `providers` array. Each provider has a `models` object.
Construct model identifiers as an object `{ providerID, modelID }`.

### Model selection workflow
- Ask user which AI model to use
- Use the model identifier in `providerID/modelID` string format (e.g., `zai-coding-plan/glm-4.7`)
- If user doesn't specify: use the default model from the `default` field in the response




## Project management

### Get current project
```
GET /project/current
Response: Project (id, name, path, etc.)
```

### List projects
```
GET /project
Response: Project[]
```
- Verify you are working on the correct project before creating sessions.



## Agent (mode) control

### Oh-my-opencode agents

**Sisyphus** (default agent)
- General purpose coding and analysis
- Use when user doesn't specify a specific agent
- Handles most coding tasks
- Can switch between planning and execution as needed

**Prometheus** (planning)
- Dedicated planning and architecture agent
- Creates detailed step-by-step plans
- **Requires special workflow**: Send `start-work` command after plan approval

### Agent selection
- Default agent: **Sisyphus** when user doesn't specify
- Specify agent via `agent` field in message body
- You can switch agents between messages without changing session


## Utility endpoints



### Get files and directories
```
GET /file?path=/relative/path
Response: FileNode[] (directory listing)
```

### Read file content
```
GET /file/content?path=/relative/path
Response: FileContent
```

### Get todo list
```
GET /session/:id/todo
Response: Todo[]
```

### Get session diff
```
GET /session/:id/diff?messageID=
Response: FileDiff[]
```
