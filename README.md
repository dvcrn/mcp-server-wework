# mcp-server-wework

MCP server for WeWork bookings and space search.

Deploy this server directly to MCP Nest:

<a href="https://mcpnest.dev/deploy?server=mcp-server-wework&package-manager=npx&env%5BWEWORK_USERNAME%5D=&env%5BWEWORK_PASSWORD%5D="><img src="https://mcpnest.dev/images/deploy-on-mcpnest.png" width="200" /></a>

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
- `bookings` — list upcoming or past bookings (times in the location timezone, with RFC 3339 and UTC timestamps)
- `book` — create bookings for one or more dates
- `quote` — get booking quotes without booking
- `info` — get detailed location information
- `me` — fetch the current user profile
- `calendar` — generate an ICS payload from bookings
- `cancel_booking` — cancel an upcoming booking by booking UUID
- `favorites` — list favorite and recent locations for a space type (0–3)
- `add_favorite` — favorite a location
- `remove_favorite` — remove a location from favorites
- `print_queue` — list the print hub queue
- `add_print_job` — upload a base64-encoded document to the print hub queue

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

