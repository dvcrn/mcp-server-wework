# mcp-server-wework

MCP server for WeWork bookings and space search.

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

## Installation

```bash
npm install -g mcp-server-wework
```

## Local development

```bash
mise install
mise run test
mise run build
./dist/mcp-server-wework
```

## Example MCP config

```json
{
  "mcpServers": {
    "wework": {
      "command": "mcp-server-wework",
      "env": {
        "WEWORK_USERNAME": "your-email@example.com",
        "WEWORK_PASSWORD": "your-password"
      }
    }
  }
}
```
