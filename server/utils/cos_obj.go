package utils

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"os"

	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

var Client *cos.Client

func InitCos() {
	u, _ := url.Parse("https://20030914132-1317776140.cos.ap-chongqing.myqcloud.com")
	b := &cos.BaseURL{BucketURL: u}
	SecretId := os.Getenv("SECRETID")
	SecretKey := os.Getenv("SECRETKEY")
	Client = cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  SecretId,
			SecretKey: SecretKey,
		},
	})
}

func PutObj(ctx context.Context, fh *multipart.FileHeader) (key string, url string, err error) {
	prefix := "images/"
	file, err := fh.Open()
	if err != nil {
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	rand8 := make([]byte, 4)
	_, _ = rand.Read(rand8)
	key = prefix + time.Now().Format("20060102_150405") + "_" + hex.EncodeToString(rand8) + ext

	ct := fh.Header.Get("Content-Type")
	var reader io.Reader = file
	if ct == "" {
		head := make([]byte, 512)
		n, _ := io.ReadFull(file, head)
		ct = http.DetectContentType(head[:n])
		reader = io.MultiReader(bytes.NewReader(head[:n]), file)
	}

	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: ct,
		},
	}
	if fh.Size > 0 {
		opt.ContentLength = fh.Size
	}

	resp, err := Client.Object.Put(ctx, key, reader, opt)
	if err != nil {
		return
	}

	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	url = Client.Object.GetObjectURL(key).String()
	return
}

func DeleteObject(ctx context.Context, key string) error {
	_, err := Client.Object.Delete(ctx, key)
	return err
}
