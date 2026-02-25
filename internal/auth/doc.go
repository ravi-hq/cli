// Package auth implements the OAuth 2.0 Device Authorization Grant flow
// for authenticating users with the Ravi backend.
//
// The device flow allows CLI applications to authenticate users by:
//  1. Requesting a device code from the server
//  2. Displaying a URL and user code for the user to visit
//  3. Polling for token completion while the user authenticates in browser
//  4. Storing the received tokens for future API calls
//
// This flow is ideal for CLI tools as it doesn't require the application
// to handle user credentials directly.
package auth
