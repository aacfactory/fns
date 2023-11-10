package compress

import "github.com/valyala/fasthttp"

type Compressor interface {
	Name() string
	Compress(p []byte) (out []byte, err error)
}

type GzipCompressor struct {
	level int
}

func (g *GzipCompressor) Name() string {
	return GzipName
}

func (g *GzipCompressor) Compress(p []byte) (out []byte, err error) {
	out = make([]byte, 0, 1)
	fasthttp.AppendGzipBytesLevel(out, p, g.level)
	return
}

type DeflateCompressor struct {
	level int
}

func (d *DeflateCompressor) Name() string {
	return DeflateName
}

func (d *DeflateCompressor) Compress(p []byte) (out []byte, err error) {
	out = make([]byte, 0, 1)
	fasthttp.AppendDeflateBytesLevel(out, p, d.level)
	return
}

type BrotliCompressor struct {
	level int
}

func (b *BrotliCompressor) Name() string {
	return BrotliName
}

func (b *BrotliCompressor) Compress(p []byte) (out []byte, err error) {
	out = make([]byte, 0, 1)
	fasthttp.AppendBrotliBytesLevel(out, p, b.level)
	return
}
