// Interactive file uploader

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"mime"
	"context"
	"bufio"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
	versioningDemoBucket = "versioning-demo"
	metadataDemoBucket = "metadata-demo"
)

var accessKey = os.Getenv("MINIO_ACCESSKEY")
var secretKey = os.Getenv("MINIO_SECRETKEY")

func createBucketIfNotExists(ctx context.Context, client *minio.Client,
		bucketName string) {
	exists, err := client.BucketExists(ctx, bucketName);
	check(err, "Error when checking for bucket %s", bucketName);
	defaultOptions := minio.MakeBucketOptions{}
	if !exists {
		client.MakeBucket(ctx, bucketName, defaultOptions);
		log("INFO", "Created bucket %s", bucketName);
	}
}

var scanner = bufio.NewScanner(os.Stdin)

func getFileInput(save bool) string {
	fmt.Print("Paste file path or press enter to open GUI: ")
	var path string
	if (scanner.Scan()) {
		path = scanner.Text();
	}
	command := "zenity"
	args := []string{"--file-selection"}
	if save {
		args = append(args, "--save")
	}
	for path == "" {
		cmd := exec.Command(command, args...);
		outb, err := cmd.Output()
		out := strings.TrimSpace(string(outb));
		if (out == "") {
			log("ERROR", "Error getting file (%v)", err)
		} else {
			path = out
		}
	}
	return path
}

func hrule() {
	fmt.Println("---------------------------------------------------");
}

func getContentType(name string) string {
	_, ext, _ := strings.Cut(name, ".")
	return mime.TypeByExtension(ext)
}

func uploadFile(ctx context.Context, client *minio.Client, bucket string, filePath string) {
	name := filepath.Base(filePath)
	cType := getContentType(name);
	info, err := client.FPutObject(ctx, bucket, name, filePath,
		minio.PutObjectOptions{ContentType: cType})
	check(err, "Failed to upload file")
	log("INFO", "Uploaded (key = %s, size = %d)", info.Key, info.Size);
}

func uploadFileToImageBucket(ctx context.Context, client *minio.Client) {
	path := getFileInput(false)
	uploadFile(ctx, client, inputImagesBucket, path)
}

var allBuckets = []string{inputImagesBucket, invertedImagesBucket, versioningDemoBucket}

func listAllBuckets(ctx context.Context, client *minio.Client) {
	for _, bucket := range allBuckets {
		fmt.Println("------", bucket, "------")
		objects := client.ListObjects(ctx, bucket,
			minio.ListObjectsOptions{WithMetadata: true, WithVersions: true});
		for object := range objects {
			fmt.Printf("%-20s | %-30v", object.Key, object.LastModified)
			if (bucket == versioningDemoBucket) {
				fmt.Printf(" | %-20v", object.VersionID)
			}
			fmt.Printf("\n")
		}
		fmt.Println()
	}
}

func downloadFile(ctx context.Context, client *minio.Client) {
	var bucket, name string
	fmt.Println("Enter bucket name & object name")
	fmt.Scanln(&bucket, &name)
	out := getFileInput(true)
	err := client.FGetObject(ctx, bucket, name, out, minio.GetObjectOptions{})
	check(err, "Cannot download object.")
}

func uploadFileToVersionedBucket(ctx context.Context, client *minio.Client) {
	path := getFileInput(false)
	uploadFile(ctx, client, versioningDemoBucket, path)
}

func uploadFileToMetadataBucket(ctx context.Context, client *minio.Client) {
	// TODO:
}

func main() {
	ctx := context.Background()
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKey, secretKey, ""),
	})
	check(err, "cannot create minio client")

	fmt.Println("Checking for buckets")
	createBucketIfNotExists(ctx, client, inputImagesBucket)
	createBucketIfNotExists(ctx, client, invertedImagesBucket)
	createBucketIfNotExists(ctx, client, versioningDemoBucket)
	client.EnableVersioning(ctx, versioningDemoBucket);

	actions := []struct {
		info string
		action func(context.Context, *minio.Client)
	}{
		{"Upload file to images", uploadFileToImageBucket},
		{"List all buckets", listAllBuckets},
		{"Upload to versioning demo bucket", uploadFileToVersionedBucket},
		{"Download an object", downloadFile},
	}

	for {
		var choice int
		for i, action := range actions {
			fmt.Printf(" | %d. %s ", i+1, action.info)
			i++
		}
		fmt.Printf(":");
		fmt.Scanln(&choice)
		if (choice >= 1 && choice <= len(actions)) {
			actions[choice - 1].action(ctx, client)
		}
	}
}
