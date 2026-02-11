package main

import (
	"flag"
	"fmt"
	"os"

	"mdm-server/internal/dep"
)

func main() {
	addr := flag.String("addr", ":8080", "Address to listen on")
	flag.Parse()

	fmt.Println("Starting Mock DEP Server...")
	fmt.Println("This server simulates Apple's Device Enrollment Program API")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  Configure your DEP client to use this URL instead of Apple's")
	fmt.Println()
	fmt.Println("Environment variable for your MDM server:")
	fmt.Printf("  DEP_MOCK_URL=http://localhost%s\n", *addr)
	fmt.Println()

	if err := dep.RunMockServer(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
