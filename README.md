# mcp-server-wework

MCP server for WeWork bookings and space search.

Deploy this server directly to MCP Nest:

[![Deploy on MCP Nest](https://mcpnest.dev/images/deploy-on-mcpnest.png)](https://mcpnest.dev/deploy?server=mcp-server-wework&package-manager=npx&env%5BWEWORK_USERNAME%5D=&env%5BWEWORK_PASSWORD%5D=)

## Install

Run it directly with npx:

```bash
npx -y mcp-server-wework
```

Or install via Go:

```bash
go install github.com/dvcrn/mcp-server-wework/cmd/mcp-server-wework@latest
```

## Usage with Claude

Add it to your MCP configuration:

```json
{
  "mcpServers": {
    "wework": {
      "command": "npx",
      "args": ["-y", "mcp-server-wework"],
      "env": {
        "WEWORK_USERNAME": "your-email@example.com",
        "WEWORK_PASSWORD": "your-password"
      }
    }
  }
}
```

## Tools

- `locations` — list WeWork locations in a city
- `desks` — list available spaces for a date
- `find_space` — alias for `desks`
- `bookings` — list upcoming or past bookings
- `book` — create bookings for one or more dates
- `quote` — get booking quotes without booking
- `info` — get detailed location information
- `me` — fetch the current user profile
- `calendar` — generate an ICS payload from bookings
- `cancel_booking` — cancel an upcoming booking by booking UUID

## Credentials

The server reads credentials from environment variables:

- `WEWORK_USERNAME`
- `WEWORK_PASSWORD`

## Local development

```bash
mise install
mise run test
mise run build
./dist/mcp-server-wework
```

