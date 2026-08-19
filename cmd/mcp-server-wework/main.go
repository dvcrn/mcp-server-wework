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
		Description: "List upcoming bookings, or past bookings with optional date filters. Times are in the booking location's timezone: date/start_time/end_time are local wall clock, timezone is the IANA zone, and starts_at/ends_at (with offset) plus starts_at_utc/ends_at_utc give unambiguous timestamps.",
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

	addTool(server, &mcp.Tool{
		Name:        "favorites",
		Description: "List the member's favorite and recent WeWork locations for a space type. space_type must be 0-3 (defaults to 0); other values are rejected by the API.",
		InputSchema: objSchema(map[string]any{
			"space_type": intSchema("Space type to query, 0-3. Defaults to 0."),
		}),
	}, func(ctx context.Context, input app.FavoritesInput) (app.FavoritesResult, error) {
		return service.Favorites(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "add_favorite",
		Description: "Favorite a WeWork location by UUID.",
		InputSchema: objSchema(map[string]any{
			"location_uuid":         strSchema("Location UUID to favorite."),
			"space_type":            intSchema("Space type, 0-3. Defaults to 0."),
			"location_type":         intSchema("Optional location type; defaults to the app value."),
			"location_account_type": intSchema("Optional location account type; defaults to the app value."),
			"reservable_uuid":       strSchema("Optional reservable UUID for a specific space."),
			"space_id":              intSchema("Optional numeric space id."),
			"inventory_name":        strSchema("Optional inventory/space name."),
			"inventory_image_url":   strSchema("Optional inventory/space image URL."),
			"floor_id":              intSchema("Optional floor id."),
		}, "location_uuid"),
	}, func(ctx context.Context, input app.FavoriteMutationInput) (any, error) {
		return service.AddFavorite(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "remove_favorite",
		Description: "Remove a WeWork location from the member's favorites. Provide location_uuid (resolved against the current favorites) or the exact hmy id.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("Location UUID to remove; resolved to the favorite's id via the current favorites list."),
			"hmy":           intSchema("Exact favorite id (the hmy value from the favorites tool). Skips lookup when provided."),
			"space_type":    intSchema("Space type, 0-3. Defaults to 0. Used when resolving location_uuid."),
		}),
	}, func(ctx context.Context, input app.FavoriteMutationInput) (any, error) {
		return service.RemoveFavorite(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "print_queue",
		Description: "List the member's WeWork print hub queue.",
		InputSchema: objSchema(map[string]any{
			"job_ids": strSchema("Optional comma-separated job ids to filter by. Defaults to the full queue."),
		}),
	}, func(ctx context.Context, input app.PrintQueueInput) (any, error) {
		return service.PrintQueue(ctx, input)
	})

	addTool(server, &mcp.Tool{
		Name:        "add_print_job",
		Description: "Upload a document to the member's WeWork print hub queue. Provide the file as base64.",
		InputSchema: objSchema(map[string]any{
			"file_name":         strSchema("File name, e.g. document.pdf."),
			"file_base64":       strSchema("Base64-encoded file contents."),
			"file_content_type": strSchema("MIME type of the file. Defaults to application/octet-stream."),
			"job_name":          strSchema("Optional job name. Defaults to the file name."),
			"copies":            intSchema("Number of copies. Defaults to 1."),
			"orientation":       strSchema("Orientation, e.g. portrait or landscape. Defaults to portrait."),
			"color_mode":        strSchema("Color mode, e.g. monochrome or color. Defaults to monochrome."),
			"sides":             strSchema("Sides, e.g. one-sided or two-sided-long-edge. Defaults to one-sided."),
			"force_media_size":  strSchema("Optional media size override."),
		}, "file_name", "file_base64"),
	}, func(ctx context.Context, input app.AddPrintJobInput) (any, error) {
		return service.AddPrintJob(ctx, input)
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

func intSchema(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}
