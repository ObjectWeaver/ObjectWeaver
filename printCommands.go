package main

import (
	"fmt"
	"os"
)

func printAscii() {
	branding := fmt.Sprintf(`
  _____ _               _     _                 
 |  ___(_)_ __ ___  ___| |__ (_)_ __ ___  _ __  
 | |_  | | '__/ _ \/ __| '_ \| | '_ ' _ \| '_ \ 
 |  _| | | | |  __/ (__| | | | | | | | | | |_) |
 |_|   |_|_|  \___|\___|_| |_|_|_| |_| |_| .__/ 
                                         |_|                                                                 `)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	developerMessage := ""
	if os.Getenv("ENVIRONMENT") == "development" {
		developerMessage = fmt.Sprintf(`
You can access the testing environment at: http://localhost:%s/
`, port)
	}

	message := fmt.Sprintf(
		"%s\n\n%s\nFirechimp License Notice\n\n"+
			"This software is provided by Firechimp (https://firechimp.ai). By using this software,\n"+
			"you automatically accept and agree to be bound by the End User License Agreement located at:\n"+
			"https://firechimp.ai/eula\n\n"+
			"For complete documentation and support, visit: https://firechimp.ai/docs/intro\n\n"+
			"© Firechimp. All rights reserved.\n\n",
		branding,
		developerMessage,
	)

	fmt.Println(message)

}
