package main

import (
	"context"
	"log"
	"strings"
	_ "time/tzdata"

	"github.com/dvcrn/mcp-server-wework/internal/app"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	service := app.NewService()
	server := mcp.NewServer(&mcp.Implementation{Name: "wework", Version: "0.1.0"}, nil)

	addTool(server, &mcp.Tool{
		Name:        "locations",
		Description: "List WeWork locations in a city.",
		InputSchema: objSchema(map[string]any{
			"city": strSchema("City name to search, e.g. New York"),
		}, "city"),
	}, func(ctx context.Context, input app.LocationsInput) (app.LocationsResult, error) {
		return service.Locations(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "desks",
		Description: "List available spaces for a date, by location UUID or city.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("One location UUID or a comma-separated list of location UUIDs."),
			"city":          strSchema("City name to search instead of location_uuid."),
			"date":          strSchema("Date in YYYY-MM-DD format. Defaults to today."),
		}),
	}, func(ctx context.Context, input app.DesksInput) (app.DesksResult, error) {
		return service.Desks(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "find_space",
		Description: "Alias for desks: find available spaces for a date in a location or city.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("One location UUID or a comma-separated list of location UUIDs."),
			"city":          strSchema("City name to search instead of location_uuid."),
			"date":          strSchema("Date in YYYY-MM-DD format. Defaults to today."),
		}),
	}, func(ctx context.Context, input app.DesksInput) (app.DesksResult, error) {
		return service.Desks(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "bookings",
		Description: "List upcoming bookings, or past bookings with optional date filters.",
		InputSchema: objSchema(map[string]any{
			"past":       boolSchema("Set true to fetch past bookings instead of upcoming bookings."),
			"start_date": strSchema("Optional start date for past bookings in YYYY-MM-DD format."),
			"end_date":   strSchema("Optional end date for past bookings in YYYY-MM-DD format."),
		}),
	}, func(ctx context.Context, input app.BookingsInput) (app.BookingsResult, error) {
		return service.Bookings(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "book",
		Description: "Book a workspace for one date, a comma-separated list of dates, or a date range like YYYY-MM-DD~YYYY-MM-DD.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("Location UUID to book."),
			"city":          strSchema("City name used together with name when location_uuid is omitted."),
			"name":          strSchema("Location name used together with city when location_uuid is omitted."),
			"date":          strSchema("A single date, comma-separated dates, or a range like 2026-04-06~2026-04-08."),
		}, "date"),
	}, func(ctx context.Context, input app.BookInput) (app.BookResults, error) {
		return service.Book(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "quote",
		Description: "Get booking quotes for one date, a comma-separated list of dates, or a date range.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("Location UUID to quote."),
			"city":          strSchema("City name used together with name when location_uuid is omitted."),
			"name":          strSchema("Location name used together with city when location_uuid is omitted."),
			"date":          strSchema("A single date, comma-separated dates, or a range like 2026-04-06~2026-04-08."),
		}, "date"),
	}, func(ctx context.Context, input app.QuoteInput) (app.QuoteResults, error) {
		return service.Quote(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "info",
		Description: "Get detailed information for a WeWork location.",
		InputSchema: objSchema(map[string]any{
			"location_uuid":  strSchema("Location UUID to inspect."),
			"city":           strSchema("City name used together with name when location_uuid is omitted."),
			"name":           strSchema("Location name used together with city when location_uuid is omitted."),
			"amenities_only": boolSchema("If true, request amenities-focused location info."),
		}),
	}, func(ctx context.Context, input app.InfoInput) (any, error) {
		return service.Info(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "me",
		Description: "Get the current user's WeWork profile. Optionally include bootstrap data.",
		InputSchema: objSchema(map[string]any{
			"include_bootstrap": boolSchema("If true, include the WeWork bootstrap payload in the result."),
		}),
	}, func(ctx context.Context, input app.MeInput) (any, error) {
		return service.Me(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "calendar",
		Description: "Generate an ICS calendar payload containing recent past and upcoming bookings.",
		InputSchema: objSchema(map[string]any{}),
	}, func(ctx context.Context, input app.CalendarInput) (app.CalendarOutput, error) {
		return service.Calendar(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "cancel_booking",
		Description: "Cancel an upcoming booking by the identifier exposed by the bookings tool.",
		InputSchema: objSchema(map[string]any{
			"booking_uuid": strSchema("The booking identifier exposed by the bookings tool."),
		}, "booking_uuid"),
	}, func(ctx context.Context, input app.CancelBookingInput) (any, error) {
		return service.CancelBooking(ctx, input)
	})

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil && !isExpectedStdioClose(err) {
		log.Fatal(err)
	}
}

func isExpectedStdioClose(err error) bool {
	return strings.Contains(err.Error(), "server is closing: EOF")
}

func addTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler func(context.Context, In) (Out, error)) {
	mcp.AddTool(server, tool, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		output, err := handler(ctx, input)
		return nil, output, err
	})
}

func objSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func strSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func boolSchema(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}
