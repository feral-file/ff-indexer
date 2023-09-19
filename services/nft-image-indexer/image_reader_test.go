package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScreenshotTags(t *testing.T) {
	// svg with <rect>
	url1 := "https://openseauserdata.com/files/dccefc262d24e3c43c1421efbc0e56f1.svg"
	buf1, err := ScreenshotTags(url1)
	assert.NotNil(t, buf1)
	assert.NoError(t, err)

	// svg without <rect>.
	url2 := "https://openseauserdata.com/files/6e3209d74c6471690348418330f76545.svg"
	buf2, err := ScreenshotTags(url2)
	assert.NotNil(t, buf2)
	assert.NoError(t, err)

	url3 := "https://assets.objkt.media/file/assets-003/QmfYFFvQ5cY7Y7H4mxmeZWmHxum8EhibDfea2ggi2PuEY8/artifact?objkt=1243&creator=tz1exNcVPJdNSqiKDxYkiZRAaxB74jq1m4CQ&viewer=tz1SidNQb9XcwP7L3MzCZD9JHmWw2ebDzgyX&danger=ignored"
	buf3, err := ScreenshotTags(url3)
	assert.NotNil(t, buf3)
	assert.NoError(t, err)

	// small size svg (< 512kb)
	url4 := "https://openseauserdata.com/files/224b81f977f415bdd7b5ffccbe28e459.svg"
	buf4, err := ScreenshotTags(url4)
	assert.NotNil(t, buf4)
	assert.NoError(t, err)
}

func TestImageReader(t *testing.T) {
	reader1 := NewURLImageReader("https://openseauserdata.com/files/6e3209d74c6471690348418330f76545.svg")
	_, mimeType, _, err := reader1.Read()

	assert.NoError(t, err)
	assert.Equal(t, "image/png", mimeType)

	reader2 := NewURLImageReader("https://openseauserdata.com/files/224b81f977f415bdd7b5ffccbe28e459.svg")
	_, mimeType, _, err = reader2.Read()

	assert.NoError(t, err)
	assert.Equal(t, "image/png", mimeType)

	reader3 := NewURLImageReader("https://i.seadn.io/gcs/files/84f6bc32f323099f62a8ef81d6b46b91.png?w=500&auto=format")
	_, mimeType, _, err = reader3.Read()

	assert.NoError(t, err)
	assert.Equal(t, "image/png", mimeType)

	reader4 := NewURLImageReader("https://gateway.autonomy.io/ipfs/QmfPLogi1UnC2KdvvtwP5VpXfm6P8tSyaU2ErLK5mnUwgK")
	_, mimeType, fileSzie, err := reader4.Read()

	assert.NoError(t, err)
	assert.Equal(t, "image/png", mimeType)
	assert.Equal(t, 1079879, fileSzie)

	reader5 := NewURLImageReader("https://ipfs.io/ipfs/QmXeQDnCCnbMwydTqoRCULSuAf9jedGY5nXYWULbzoa264")
	_, mimeType, fileSzie, err = reader5.Read()

	assert.NoError(t, err)
	assert.Equal(t, "image/jpeg", mimeType)
	assert.Equal(t, 190183, fileSzie)

	// svg without extension
	reader6 := NewURLImageReader("https://ipfs.io/ipfs/QmZ7Ck3zJFKfN1aYgrSmTTxWRoAAzE7hF4rqA5qip2RmhW")
	_, mimeType, fileSzie, err = reader6.Read()

	assert.NoError(t, err)
	assert.Equal(t, "image/png", mimeType)
	assert.Equal(t, 5926, fileSzie)
}
