// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://objectweaver.dev/licensing/server-side-public-license>.
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
