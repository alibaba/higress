# HackMD MCP Server

The MCP server implementation based on the HackMD API interacts with the HackMD platform through the MCP protocol. HackMD is a real-time, cross-platform collaborative Markdown knowledge base that allows users to co-edit documents with others on desktop, tablet, or mobile devices.

## Features

HackMD MCP Server provides the following features:

- **User Data**: Retrieve user profile information and related configurations.
    - `get_me`: Retrieve user data.

- **Note Management**: Create, read, update, and delete personal notes.
    - `get_notes`: Retrieve the user's note list.
    - `post_notes`: Create a new note.
    - `get_notes_noteId`: Retrieve a specific note by its ID.
    - `patch_notes_noteId`: Update the content of a note.
    - `delete_notes_noteId`: Delete a note.

- **Team Collaboration**: Manage team-related notes.
    - `get_teams`: Retrieve the list of teams the user participates in.
    - `get_teams_teamPath_notes`: Retrieve the list of notes in a team.
    - `patch_teams_teamPath_notes_noteId`: Update the content of a note within a team.
    - `delete_teams_teamPath_notes_noteId`: Delete a note from a team.

- **Browsing History**: View the user's browsing history.
    - `get_history`: Retrieve the user's browsing history.

## Usage Guide

### Get AccessToken

参考 [HackMD API 文档](https://hackmd.io/@hackmd-api/developer-portal/https%3A%2F%2Fhackmd.io%2F%40hackmd-api%2FrkoVeBXkq) 获取 AccessToken。

### Generate SSE URL

On the MCP Server interface, log in and enter the AccessToken to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

``` json
"mcpServers": {
    "hackmd": {
      "url": "https://mcp.higress.ai/mcp-hackmd/{generate_key}",
    }
}
```
