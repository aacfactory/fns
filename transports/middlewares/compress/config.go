package compress

type Config struct {
	Enable       bool   `json:"enable"`
	Default      string `json:"default"`
	GzipLevel    int    `json:"gzipLevel"`
	DeflateLevel int    `json:"deflateLevel"`
	BrotliLevel  int    `json:"brotliLevel"`
}
