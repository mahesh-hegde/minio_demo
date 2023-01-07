// Inverts them and puts them into inverted-images bucket

package main

import (
	"fmt"
	"os"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/disintegration/imaging"
)

func check(e error, format string, v ...any) {
	if (e != nil) {
		fmt.Fprintln(os.Stderr, "Error:", e);
		if (format != "") {
			fmt.Fprintf(os.Stderr, format+"\n", v...);
		}
	}
}

func log(level string, message string, args ...any) {
	const colorReset = "\033[0m"
	const colorYellow = "\033[33m"
	const colorRed = "\033[31m"
	color := colorYellow
	if (level == "ERROR") {
		color = colorRed
	}
	fmt.Fprint(os.Stderr, color);
	fmt.Fprintf(os.Stderr, "%-8s", level);
	fmt.Fprintf(os.Stderr, message+"\n", args...);
	fmt.Fprint(os.Stderr, colorReset);
}

const (
	endpoint = "localhost:9000"
	inputImagesBucket = "input-images"
	invertedImagesBucket = "inverted-images"
)

var accessKey = os.Getenv("MINIO_ACCESSKEY")
var secretKey = os.Getenv("MINIO_SECRETKEY")

func tempName() string {
	tempFile, err := os.CreateTemp(".", "image_listener_temp_")
	check(err, "cannot create temporary file")
	name := tempFile.Name();
	tempFile.Close();
	return name
}

func main() {

	if accessKey == "" || secretKey == "" {
		log("ERROR", "please set MINIO_ACCESSKEY and MINIO_SECRETKEY")
		return
	}

	ctx := context.Background()
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKey, secretKey, ""),
	})
	check(err, "cannot create minio client")

	fmt.Println("Starting bucket listener")
	eventSpec := []string{"s3:ObjectCreated:*"}
	channel := client.ListenBucketNotification(ctx, inputImagesBucket, "", "", eventSpec)
	for event := range channel {
		if (event.Err != nil) {
			log("ERROR", event.Err.Error())
		}
		for _, record := range event.Records {
			key := record.S3.Object.Key
			mimetype := record.S3.Object.ContentType
			var suffix string
			if (mimetype == "image/jpeg") {
				suffix = ".jpg"
			} else if (mimetype == "image/png") {
				suffix = ".png"
			} else {
				log("ERROR", "Not a JPEG or PNG file")
				continue;
			}
			tempPath := tempName()
			defer os.Remove(tempPath);
			processedPath := tempPath + "_inverted" + suffix
			defer os.Remove(processedPath)

			client.FGetObject(ctx, inputImagesBucket, key, tempPath, minio.GetObjectOptions{})
			image, err := imaging.Open(tempPath)
			check(err, "cannot open temporary file")
			inverted := imaging.Invert(image)
			err = imaging.Save(inverted, processedPath);
			check(err, "cannot save image")
			info, err := client.FPutObject(ctx, invertedImagesBucket, key, processedPath, minio.PutObjectOptions{})
			check(err, "CANNOT UPLOAD OBJECT")
			log("INFO", "Uploaded %s (size: %d)", info.Key, info.Size)
		}
	}
}
