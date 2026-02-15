# OpenClaw OpenCode Skills

A collection of skills for OpenCode agents, designed to enhance their capabilities and interaction with various systems.

## Available Skills

### [opencode-web-skill](./opencode-web-skill)

**Description**: Controls OpenCode's agents (Sisyphus, Prometheus, Atlas) via the web API using a robust **Daemon-Client architecture** implemented in Go.

**Key Features**:
-   **Go-based Binary**: Uses `opencode_skill` for fast and reliable interactions.
-   **Session Management**: Supports persistent sessions with project and session names.
-   **Daemon-Client Architecture**: Ensures background processing and non-blocking command submissions.
-   **Multiple Agents**: Switch between Sisyphus, Prometheus, and Atlas agents easily.

**Documentation**: See [opencode-web-skill/SKILL.md](./opencode-web-skill/SKILL.md) for detailed usage instructions.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
