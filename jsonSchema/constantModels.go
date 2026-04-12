package jsonSchema

type DataType string

const (
	Object  DataType = "object"
	Number  DataType = "number"
	Integer DataType = "integer"
	String  DataType = "string"
	Array   DataType = "array"
	Null    DataType = "null"
	Boolean DataType = "boolean"
	Map     DataType = "map"
	Byte    DataType = "byte" //this will be used for the audio and image data selection (if this is selected as byte then either Image or Audio must not be nil, if it is then nothing will occur and an empty byte will be returned. The same is true if both are filled.
	Vector  DataType = "vector"
)