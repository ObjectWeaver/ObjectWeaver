package jsonSchema

const (
	//this code is nicked from go-openai
	CreateImageSize256x256   = "256x256"
	CreateImageSize512x512   = "512x512"
	CreateImageSize1024x1024 = "1024x1024"

	// dall-e-3 supported only.
	CreateImageSize1792x1024 = "1792x1024"
	CreateImageSize1024x1792 = "1024x1792"
)

// Image if you want the Url of the image use the DataType String otherwise use the DataType Byte
type Image struct {
	Model string `json:"model,omitempty"`
	Size  string  `json:"size,omitempty"`
}

type SendImage struct {
	ImagesData [][]byte `json:"imagesData,omitempty"` //When sending multiple images take into account the model you have selected. Such that Gemini Models support multiple images whereas the Claude models only support one image at a time
}
