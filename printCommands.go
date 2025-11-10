package main

import (
	"fmt"
	"os"
)

func printAscii() {
	branding := fmt.Sprintf(`      
   ___  _     _           _ __        __                        
  / _ \| |__ (_) ___  ___| |\ \      / /__  __ ___   _____ _ __ 
 | | | | '_ \| |/ _ \/ __| __\ \ /\ / / _ \/ _' \ \ / / _ \ '__|
 | |_| | |_) | |  __/ (__| |_ \ V  V /  __/ (_| |\ V /  __/ |   
  \___/|_.__// |\___|\___|\__| \_/\_/ \___|\__,_| \_/ \___|_|   
           |__/                                                 
	`)

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
		"%s\n\n%s\nObjectWeaver License Notice\n\n"+
			"This software is provided by ObjectWeaver (https://objectweaver.dev). By using this software,\n"+
			"you automatically accept and agree to be bound by the License Agreement located at:\n"+
			"https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt\n\n"+
			"For complete documentation and support, visit: https://objectweaver.dev/docs/intro\n\n"+
			"© ObjectWeaver. All rights reserved.\n\n",
		branding,
		developerMessage,
	)

	fmt.Println(message)

}
