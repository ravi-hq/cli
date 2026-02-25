// Package cli defines the Cobra command structure for the Ravi CLI.
//
// Commands are organized hierarchically:
//   - root: Base command with global flags (--json)
//   - auth: Authentication subcommands (login, logout, status)
//   - inbox: Message viewing subcommands (list, email, sms)
//
// All commands respect the --json flag for machine-parseable output
// and use the output package formatters for consistent display.
package cli
