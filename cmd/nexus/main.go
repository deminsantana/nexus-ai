package main

import "nexus-core/internal/cli"

func main() {
	cli.Execute() // Make sure the function is exported as 'Execute' in the cli package, or replace with the correct exported function name, e.g., cli.Run()
}
