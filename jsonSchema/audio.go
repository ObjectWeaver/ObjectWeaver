package jsonSchema

// TextToSpeech the DataType to use with this type is Byte
type TextToSpeech struct {
	Model         string `json:"model,omitempty"`
	StringToAudio string `json:"stringToAudio,omitempty"`
	Voice         string            `json:"voice,omitempty"`
	Format        string            `json:"format,omitempty"`
}

const Text = "text"
const SRT = "srt"
const VTT = "vtt"
const JSON = "json"
const VerboseJSON = "verbose-json"


// SpeechToText the DataType to use with this type is String
type SpeechToText struct {
	Model             string `json:"model,omitempty"`
	AudioToTranscribe []byte            `json:"audioToTranscribe,omitempty"`
	Language          string            `json:"language,omitempty"` //must be in the format of ISO-639-1  will default to en (english)
	Format            string            `json:"format,omitempty"`
	ToString          bool              `json:"toString,omitempty"`
	ToCaptions        bool              `json:"toCaptions,omitempty"`
	ChunkingStrategy  string            `json:"chunkingStrategy,omitempty"`
	ExtraBody		map[string]any    `json:"extraBody,omitempty"`
	
}
