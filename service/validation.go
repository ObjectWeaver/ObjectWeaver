package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/objectweaver/go-sdk/client"
	"github.com/objectweaver/go-sdk/jsonSchema"
)

type UserTier string

const (
	Free       UserTier = "free"
	Pro        UserTier = "pro"
	Startup    UserTier = "startup"
	Enterprise UserTier = "enterprise"
)

func ValidateDefinitionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}

		// Log body size
		//log.Printf("Request body size: %d bytes", len(bodyBytes))

		// Reset the request body so it can be read again
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Decode the request body
		var requestBody client.RequestBody
		if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Log decoded body
		//bodyLog, _ := json.Marshal(requestBody)
		//log.Printf("Decoded request body: %s", bodyLog)

		// Perform validation
		userTier := os.Getenv("USER_TIER")
		if requestBody.Definition != nil {
			if err := validateDefinition(requestBody.Definition, UserTier(userTier), 0); err != nil {
				log.Printf("Validation error: %v", err)
				http.Error(w, fmt.Sprintf("Invalid definition: %v", err), http.StatusBadRequest)
				return
			}
		}

		// Pass request to the next handler
		next.ServeHTTP(w, r)
	})
}

func validateDefinition(def *jsonSchema.Definition, tier UserTier, depth int) error {
	// Log the current definition and depth
	//log.Printf("Validating definition at depth %d: %+v", depth, def)

	// Check if the depth of properties exceeds allowed depth for Free tier users
	if depth > 2 && tier == Free {
		log.Printf("Validation Error: Free tier users are only allowed to have properties one layer deep. Current depth: %d", depth)
		return errors.New("free users can only have properties one layer deep")
	}

	// Recursively validate each property in the definition
	for key, prop := range def.Properties {
		if depth > 10 { // Arbitrary large number to prevent infinite recursion
			log.Printf("Validation Error: Recursive depth exceeds safe limit. Current depth: %d", depth)
			return errors.New("recursion depth exceeds safe limit")
		}

		// Recursively validate the property
		if err := validateDefinition(&prop, tier, depth+1); err != nil {
			log.Printf("Error validating property %s: %v", key, err)
			return fmt.Errorf("property %s: %v", key, err)
		}
	}

	// Perform tier-specific validations
	switch tier {
	case Free:
		if len(def.Properties) > 0 && depth > 2 {
			log.Printf("Validation Error: Free tier users cannot have properties more than one layer deep. Depth: %d, Properties found: %d", depth, len(def.Properties))
			return errors.New("free users cannot have properties more than one layer deep")
		}
		if def.Items != nil {
			log.Printf("Validation Error: Free tier users are not allowed to use the 'Items' field.")
			return errors.New("free users cannot use Items field")
		}
		if def.ProcessingOrder != nil {
			log.Printf("Validation Error: Free tier users are not allowed to use the 'ProcessingOrder' field.")
			return errors.New("free users cannot use ProcessingOrder field")
		}
		if def.SystemPrompt != nil {
			log.Printf("Validation Error: Free tier users are not allowed to use the 'SystemPrompt' field.")
			return errors.New("free users cannot use SystemPrompt field")
		}
		if def.Choices != nil {
			log.Printf("Validation Error: Free tier users are not allowed to use the 'Choices' field.")
			return errors.New("free users cannot use Choices field")
		}
		if def.Req != nil {
			log.Printf("Validation Error: Free tier users are not allowed to use the 'Req' field.")
			return errors.New("free users cannot use Req field")
		}

	case Pro:
		if def.Choices != nil {
			log.Printf("Validation Error: Pro tier users are not allowed to use the 'Choices' field.")
			return errors.New("Pro users cannot use Choices field")
		}
		if def.Req != nil {
			log.Printf("Validation Error: Pro tier users are not allowed to use the 'Req' field.")
			return errors.New("Pro users cannot use Req field")
		}

	case Startup:
		// No restrictions for Startup tier in the current implementation
		return nil
	case Enterprise:
		// No restrictions for Enterprise tier in the current implementation
		return nil
	default:
		log.Printf("Validation Error: Unknown user tier '%v' encountered.", tier)
		return errors.New("unknown user tier")
	}

	return nil
}
