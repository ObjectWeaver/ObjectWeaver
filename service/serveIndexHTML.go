package service

import (
	"bytes"
	"net/http"
	"os"
	"text/template"
)

func ServeIndexHTML(w http.ResponseWriter, r *http.Request) {
	// Define the file path to index.html
	filePath := "/static/index.html"

	// Read the HTML file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Define environment variables
	authToken := os.Getenv("PASSWORD")

	// Create a template with environment variables
	tmpl, err := template.New("index").Parse(string(fileContent))
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	// Define the data to pass to the template
	data := struct {
		AuthToken string
	}{
		AuthToken: authToken,
	}

	// Execute the template with the data
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}

	// Set the Content-Type header and write the response
	w.Header().Set("Content-Type", "text/html")
	_, err = w.Write(buf.Bytes())
	if err != nil {
		http.Error(w, "Error writing response", http.StatusInternalServerError)
		return
	}
}
