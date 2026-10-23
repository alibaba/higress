# GitHub MCP Server

An MCP server implementation of the GitHub API, supporting file operations, repository management, search, and more.

Source code: [https://github.com/modelcontextprotocol/servers/tree/main/src/github](https://github.com/modelcontextprotocol/servers/tree/main/src/github)

## Features

- **Automatic branch creation**: Automatically creates branches if they don't exist when creating/updating files or pushing changes
- **Comprehensive error handling**: Provides clear error messages for common issues
- **Git history preservation**: Operations preserve complete Git history, no force pushing
- **Batch operations**: Supports both single file and batch file operations
- **Advanced search**: Supports code, issues/PRs, and user search

## Usage Guide

### Get AccessToken
[Create GitHub personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens):
   1. Visit [Personal access tokens](https://github.com/settings/tokens) (in GitHub Settings > Developer settings)
   2. Select repositories the token can access (public, all, or selected)
   3. Create token with `repo` permissions ("Full control of private repositories")
      - Or, if only using public repositories, select only `public_repo` permissions
   4. Copy the generated token
   
### Generate SSE URL

On the MCP Server interface, log in and enter the AccessToken to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "github": {
      "url": "https://mcp.higress.ai/mcp-github/{generate_key}",
    }
}
```
