package main

import (
	"context"
	"encoding/json"
	"log"
	_ "time/tzdata"

	"github.com/dvcrn/mcp-server-wework/internal/app"
	"github.com/dvcrn/mcp-server-wework/internal/mcp"
)

func main() {
	service := app.NewService()
	server := mcp.NewServer("wework", "0.1.0")

	server.AddTool(mcp.Tool{
		Name:        "locations",
		Description: "List WeWork locations in a city.",
		InputSchema: objSchema(map[string]any{
			"city": strSchema("City name to search, e.g. New York"),
		}, "city"),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.LocationsInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Locations(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "desks",
		Description: "List available spaces for a date, by location UUID or city.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("One location UUID or a comma-separated list of location UUIDs."),
			"city":          strSchema("City name to search instead of location_uuid."),
			"date":          strSchema("Date in YYYY-MM-DD format. Defaults to today."),
		}),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.DesksInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Desks(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "find_space",
		Description: "Alias for desks: find available spaces for a date in a location or city.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("One location UUID or a comma-separated list of location UUIDs."),
			"city":          strSchema("City name to search instead of location_uuid."),
			"date":          strSchema("Date in YYYY-MM-DD format. Defaults to today."),
		}),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.DesksInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Desks(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "bookings",
		Description: "List upcoming bookings, or past bookings with optional date filters.",
		InputSchema: objSchema(map[string]any{
			"past":       boolSchema("Set true to fetch past bookings instead of upcoming bookings."),
			"start_date": strSchema("Optional start date for past bookings in YYYY-MM-DD format."),
			"end_date":   strSchema("Optional end date for past bookings in YYYY-MM-DD format."),
		}),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.BookingsInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Bookings(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "book",
		Description: "Book a workspace for one date, a comma-separated list of dates, or a date range like YYYY-MM-DD~YYYY-MM-DD.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("Location UUID to book."),
			"city":          strSchema("City name used together with name when location_uuid is omitted."),
			"name":          strSchema("Location name used together with city when location_uuid is omitted."),
			"date":          strSchema("A single date, comma-separated dates, or a range like 2026-04-06~2026-04-08."),
		}, "date"),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.BookInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Book(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "quote",
		Description: "Get booking quotes for one date, a comma-separated list of dates, or a date range.",
		InputSchema: objSchema(map[string]any{
			"location_uuid": strSchema("Location UUID to quote."),
			"city":          strSchema("City name used together with name when location_uuid is omitted."),
			"name":          strSchema("Location name used together with city when location_uuid is omitted."),
			"date":          strSchema("A single date, comma-separated dates, or a range like 2026-04-06~2026-04-08."),
		}, "date"),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.QuoteInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Quote(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "info",
		Description: "Get detailed information for a WeWork location.",
		InputSchema: objSchema(map[string]any{
			"location_uuid":  strSchema("Location UUID to inspect."),
			"city":           strSchema("City name used together with name when location_uuid is omitted."),
			"name":           strSchema("Location name used together with city when location_uuid is omitted."),
			"amenities_only": boolSchema("If true, request amenities-focused location info."),
		}),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.InfoInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Info(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "me",
		Description: "Get the current user's WeWork profile. Optionally include bootstrap data.",
		InputSchema: objSchema(map[string]any{
			"include_bootstrap": boolSchema("If true, include the WeWork bootstrap payload in the result."),
		}),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.MeInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Me(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "calendar",
		Description: "Generate an ICS calendar payload containing recent past and upcoming bookings.",
		InputSchema: objSchema(map[string]any{}),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.CalendarInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.Calendar(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "cancel_booking",
		Description: "Cancel an upcoming booking by booking UUID. Optional override fields are available if WeWork's cancellation payload needs adjustment.",
		InputSchema: objSchema(map[string]any{
			"booking_uuid":          strSchema("The UUID of the upcoming booking to cancel."),
			"booking_id":            strSchema("Optional override for bookingId in the cancellation request."),
			"booking_location_type": intSchema("Optional override for bookingLocationType. Defaults to the booking's sourceType or 0."),
			"reservable_id":         strSchema("Optional override for reservableId."),
			"location_id":           strSchema("Optional override for locationId."),
			"booking_type":          intSchema("Optional override for bookingType. 0=conference room, 2=private office, 4=shared workspace."),
			"reservation_id":        strSchema("Optional override for reservationId."),
			"is_on_demand":          boolSchema("Optional override for the isOnDemand query param. Defaults to false."),
			"platform_type":         intSchema("Optional override for platFormType. Defaults to 2 (iOS app)."),
		}, "booking_uuid"),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.CancelBookingInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.CancelBooking(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "print_queue",
		Description: "Fetch the WeWork print hub queue. job_ids defaults to 0, which returns the current queue.",
		InputSchema: objSchema(map[string]any{
			"job_ids": strSchema("Comma-separated print job IDs to fetch. Defaults to 0."),
		}),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.PrintQueueInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.PrintQueue(ctx, input)
		},
	})

	server.AddTool(mcp.Tool{
		Name:        "add_print_job",
		Description: "Upload a local file to the WeWork print hub queue.",
		InputSchema: objSchema(map[string]any{
			"file_path":             strSchema("Absolute or relative path to the local file to upload."),
			"copies":                intSchema("Number of copies. Defaults to 1."),
			"force_media_size":      strSchema("Printer media size value. Defaults to null."),
			"orientation_requested": strSchema("Print orientation, e.g. portrait or landscape. Defaults to portrait."),
			"print_color_mode":      strSchema("Print color mode, e.g. monochrome or color. Defaults to monochrome."),
			"sides":                 strSchema("Duplex setting, e.g. one-sided or two-sided-long-edge. Defaults to one-sided."),
			"job_name":              strSchema("Optional print job name. Defaults to the file name."),
			"content_type":          strSchema("Optional MIME type override. Defaults from the file extension."),
		}, "file_path"),
		Handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var input app.AddPrintJobInput
			if err := decode(raw, &input); err != nil {
				return nil, err
			}
			return service.AddPrintJob(ctx, input)
		},
	})

	if err := mcp.Run(context.Background(), server); err != nil {
		log.Fatal(err)
	}
}

func decode[T any](raw json.RawMessage, out *T) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}
	return nil
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
