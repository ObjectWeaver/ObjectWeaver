package responseCleaner

type IResponseCleaner interface {
	Clean(response, key string) string
}
