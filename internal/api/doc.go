// Package api provides the HTTP client and type definitions for communicating
// with the Ravi backend API.
//
// The package includes:
//   - Client: HTTP client with automatic token refresh and authentication
//   - Auth functions: Device code flow for OAuth 2.0 authentication
//   - Inbox functions: Fetching email threads and SMS conversations
//   - Type definitions: Request/response structures for all API endpoints
//
// Example usage:
//
//	client, err := api.NewClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	threads, err := client.ListEmailThreads(false)
package api
