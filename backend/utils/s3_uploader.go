package utils

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	awsconfig "rbac/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Uploader struct {
	client            *s3.Client
	bucket            string
	folder            string
	maxSizeBytes      int64
	compressThreshold int64
	quality           int
}

func NewS3Uploader(cfg *awsconfig.Config) (ImageUploader, error) {
	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWS.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg)

	return &S3Uploader{
		client:            s3Client,
		bucket:            cfg.AWS.Bucket,
		folder:            cfg.AWS.Folder,
		maxSizeBytes:      cfg.Image.MaxSizeBytes,
		compressThreshold: cfg.Image.CompressThresholdBytes,
		quality:           cfg.Image.Quality,
	}, nil
}

// Upload uploads file to S3, compressing if necessary
func (u *S3Uploader) Upload(file *multipart.FileHeader) (string, error) {
	if file.Size > u.maxSizeBytes {
		return "", fmt.Errorf("file size (%d bytes) exceeds maximum (%d bytes)", file.Size, u.maxSizeBytes)
	}

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	fileBytes := make([]byte, file.Size)
	if _, err := src.Read(fileBytes); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	contentType, err := DetectImageContentType(fileBytes)
	if err != nil {
		return "", fmt.Errorf("unsupported image type")
	}
	if err := ValidateDecodableImage(fileBytes); err != nil {
		return "", err
	}

	return u.UploadValidated(fileBytes, contentType)
}

// UploadValidated uploads sniffed image bytes with a random safe filename.
func (u *S3Uploader) UploadValidated(fileBytes []byte, contentType string) (string, error) {
	if int64(len(fileBytes)) > u.maxSizeBytes {
		return "", fmt.Errorf("file size (%d bytes) exceeds maximum (%d bytes)", len(fileBytes), u.maxSizeBytes)
	}

	var uploadBytes []byte
	var finalFilename string
	ext := SafeUploadFilename(contentType)

	if int64(len(fileBytes)) > u.compressThreshold {
		log.Printf("[S3_UPLOAD] Compressing image (size: %d bytes)", len(fileBytes))
		compressed, err := u.compressImage(fileBytes, "upload"+ext)
		if err != nil {
			return "", fmt.Errorf("invalid image content")
		}
		uploadBytes = compressed
		finalFilename = fmt.Sprintf("%d_compressed%s", time.Now().UnixNano(), ext)
		log.Printf("[S3_UPLOAD_COMPRESSED] Original: %d bytes -> Compressed: %d bytes", len(fileBytes), len(uploadBytes))
	} else {
		uploadBytes = fileBytes
		finalFilename = fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		log.Printf("[S3_UPLOAD] Uploading without compression (size: %d bytes)", len(fileBytes))
	}

	s3Key := fmt.Sprintf("%s/%s", u.folder, finalFilename)

	uploader := manager.NewUploader(u.client)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(uploadBytes),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Printf("[S3_UPLOAD_SUCCESS] Key: %s, Location: %s", s3Key, result.Location)
	return result.Location, nil
}

// compressImage compresses an image to reduce file size
func (u *S3Uploader) compressImage(imageBytes []byte, filename string) ([]byte, error) {
	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Encode with compression
	var out bytes.Buffer
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		err = jpeg.Encode(&out, img, &jpeg.Options{Quality: u.quality})
	case "png":
		enc := &png.Encoder{CompressionLevel: png.DefaultCompression}
		err = enc.Encode(&out, img)
	default:
		// For other formats, encode as JPEG
		err = jpeg.Encode(&out, img, &jpeg.Options{Quality: u.quality})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return out.Bytes(), nil
}

// GenerateAuthToken is deprecated (was for ImageKit)
func (u *S3Uploader) GenerateAuthToken() (*AuthTokenResponse, error) {
	return nil, fmt.Errorf("GenerateAuthToken is not supported for S3 uploader. Use Upload() instead")
}

// addCompressionSuffix adds a suffix to indicate the file was compressed
func addCompressionSuffix(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	return fmt.Sprintf("%s_compressed%s", name, ext)
}
