package controllers

import (
	// "backend/config"
	// "backend/models"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	// "log"
	"mime/multipart"
	// "net/http"

	// "net/http"
	"os"
	"path/filepath"

	// "strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nfnt/resize"

)

// // Ограничения по размерам файлов
// const maxFileSize = 5 * 1024 * 1024    // 5MB
// const compressThreshold = 1 * 1024 * 1024 // 1MB

// SaveProductPhoto сохраняет фото товара и возвращает только имя файла
func SaveProductPhoto(c *gin.Context, file *multipart.FileHeader, productID string) (string, error) {
	// Проверка размера файла
	if file.Size > maxFileSize {
		return "", fmt.Errorf("file size exceeds the 5MB limit")
	}

	// // Проверка расширения файла
	// allowedExtensions := map[string]bool{
	// 	".jpg":  true,
	// 	".jpeg": true,
	// 	".png":  true,
	// }

	fileExt := strings.ToLower(filepath.Ext(file.Filename))
	// if !allowedExtensions[fileExt] {
	// 	return "", fmt.Errorf("unsupported file format: %s", fileExt)
	// }

	// Директория для сохранения фото
	productDir := "./uploads/products"
	if _, err := os.Stat(productDir); os.IsNotExist(err) {
		if err := os.MkdirAll(productDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create product directory: %v", err)
		}
	}

	// Генерация уникального имени файла
	filename := fmt.Sprintf("%s_%d%s", productID, time.Now().Unix(), fileExt)
	fullPath := filepath.Join(productDir, filename)

	// Открытие загруженного файла
	srcFile, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer srcFile.Close()

	// Сжимаем фото, если его размер больше 1MB
	if file.Size > compressThreshold {
		var img image.Image
		if fileExt == ".png" {
			img, err = png.Decode(srcFile)
		} else {
			img, err = jpeg.Decode(srcFile)
		}
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %v", err)
		}

		// Изменяем размер изображения до 50% от его оригинального размера
		compressedImg := resize.Resize(800, 0, img, resize.Lanczos3)

		// Сохраняем сжатое изображение
		outFile, err := os.Create(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to create file: %v", err)
		}
		defer outFile.Close()

		// Сохранение сжатого изображения в JPEG формате
		err = jpeg.Encode(outFile, compressedImg, &jpeg.Options{Quality: 80})
		if err != nil {
			return "", fmt.Errorf("failed to save compressed image: %v", err)
		}
	} else {
		// Если изображение меньше 1MB, просто сохраняем его без сжатия
		if err := c.SaveUploadedFile(file, fullPath); err != nil {
			return "", fmt.Errorf("failed to save product photo: %v", err)
		}
	}

	return filename, nil
}


// SaveCategoryPhoto сохраняет фото категории и возвращает только имя файла
func SaveCategoryPhoto(c *gin.Context, file *multipart.FileHeader, categoryID string) (string, error) {
	// Проверка размера файла
	if file.Size > maxFileSize {
		return "", fmt.Errorf("file size exceeds the 5MB limit")
	}

	// Проверка расширения файла
	fileExt := strings.ToLower(filepath.Ext(file.Filename))

	// Директория для сохранения фото
	categoryDir := "./uploads/categories"
	if _, err := os.Stat(categoryDir); os.IsNotExist(err) {
		if err := os.MkdirAll(categoryDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create category directory: %v", err)
		}
	}

	// Генерация уникального имени файла
	filename := fmt.Sprintf("%s_%d%s", categoryID, time.Now().Unix(), fileExt)
	fullPath := filepath.Join(categoryDir, filename)

	// Открытие загруженного файла
	srcFile, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer srcFile.Close()

	// Сжимаем фото, если его размер больше 1MB
	if file.Size > compressThreshold {
		var img image.Image
		if fileExt == ".png" {
			img, err = png.Decode(srcFile)
		} else {
			img, err = jpeg.Decode(srcFile)
		}
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %v", err)
		}

		// Изменяем размер изображения до 50% от его оригинального размера
		compressedImg := resize.Resize(800, 0, img, resize.Lanczos3)

		// Сохраняем сжатое изображение
		outFile, err := os.Create(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to create file: %v", err)
		}
		defer outFile.Close()

		// Сохранение сжатого изображения в JPEG формате
		err = jpeg.Encode(outFile, compressedImg, &jpeg.Options{Quality: 80})
		if err != nil {
			return "", fmt.Errorf("failed to save compressed image: %v", err)
		}
	} else {
		// Если изображение меньше 1MB, просто сохраняем его без сжатия
		if err := c.SaveUploadedFile(file, fullPath); err != nil {
			return "", fmt.Errorf("failed to save category photo: %v", err)
		}
	}

	return filename, nil
}


const (
	maxFileSize       = 5 * 1024 * 1024
	compressThreshold = 100  * 1024
	maxImageWidth     = 1500
	maxImageHeight    = 1500
	previewSize       = 300
	s3Endpoint        = "s3.ru1.storage.beget.cloud"
	s3AccessKey       = "QSJBZ1JPEDIY779JC539"
	s3SecretKey       = "gFxzngf9mT3jNl8ABsnCnOtrCYUFo9Q3jePCUDMk"
	s3Bucket          = "c335b5a303c2-muf56"
	cdnDomain         = "storage.nadim.shop"
)

var s3Client *minio.Client

func init() {
	client, err := minio.New(s3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
		Secure: true,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize S3 client: %v", err))
	}
	s3Client = client
}
func SaveProductPhotoToS3(c *gin.Context, file *multipart.FileHeader, productID string) (string, string, error) {
	if file.Size > maxFileSize {
		return "", "", fmt.Errorf("file size exceeds the 5MB limit")
	}

	fileExt := strings.ToLower(filepath.Ext(file.Filename))
	baseName := fmt.Sprintf("products/%s_%d", productID, time.Now().Unix())
	mainFilename := fmt.Sprintf("%s%s", baseName, fileExt)
	previewFilename := fmt.Sprintf("%s_preview%s", baseName, fileExt)

	srcFile, err := file.Open()
	if err != nil {
		return "", "", fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer srcFile.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		return "", "", fmt.Errorf("unsupported file format: %s", contentType)
	}

	// Read full image bytes for decode and re-use
	originalData, err := io.ReadAll(srcFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to read image data: %v", err)
	}
	srcReader := bytes.NewReader(originalData)

	var img image.Image
	if contentType == "image/png" {
		img, err = png.Decode(srcReader)
	} else {
		img, err = jpeg.Decode(srcReader)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to decode image: %v", err)
	}

	var bufMain bytes.Buffer
	if file.Size >= compressThreshold {
		resizedMain := resize.Resize(800, 0, img, resize.Lanczos3)
		err = jpeg.Encode(&bufMain, resizedMain, &jpeg.Options{Quality: 80})
		if err != nil {
			return "", "", fmt.Errorf("failed to encode resized image: %v", err)
		}
	} else {
		bufMain.Write(originalData)
	}

	_, err = s3Client.PutObject(context.Background(), s3Bucket, mainFilename, &bufMain, int64(bufMain.Len()), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload main image to S3: %v", err)
	}

	previewImg := resize.Thumbnail(previewSize, previewSize, img, resize.Lanczos3)
	var bufPreview bytes.Buffer
	err = jpeg.Encode(&bufPreview, previewImg, &jpeg.Options{Quality: 75})
	if err != nil {
		return "", "", fmt.Errorf("failed to encode preview image: %v", err)
	}
	_, err = s3Client.PutObject(context.Background(), s3Bucket, previewFilename, &bufPreview, int64(bufPreview.Len()), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload preview image to S3: %v", err)
	}

	mainURL := fmt.Sprintf("https://%s/%s", cdnDomain, mainFilename)
	previewURL := fmt.Sprintf("https://%s/%s", cdnDomain, previewFilename)
	return mainURL, previewURL, nil
}
// UploadCategoryPhotoToS3 сохраняет категорию в S3 и возвращает полный путь URL
func UploadCategoryPhotoToS3(file *multipart.FileHeader, categoryID string) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer f.Close()

	size := file.Size

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	mainFilename := fmt.Sprintf("categories/%s%s", categoryID, ext)
	_, err = s3Client.PutObject(context.Background(), s3Bucket, mainFilename, f, size, minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}

	return fmt.Sprintf("https://%s/%s", cdnDomain, mainFilename), nil
}

// func MigrateLocalProductImages() {
// 	if config.ProductTemplateCollection == nil || config.ProductCollection == nil {
// 		log.Println("[Migration] Mongo collections not initialized")
// 		return
// 	}

// 	cursor, err := config.ProductTemplateCollection.Find(context.TODO(), bson.M{})
// 	if err != nil {
// 		log.Println("Failed to retrieve product templates")
// 		return
// 	}
// 	defer cursor.Close(context.TODO())

// 	migrated := 0
// 	skipped := 0

// 	for cursor.Next(context.TODO()) {
// 		var product models.ProductTemplate
// 		if err := cursor.Decode(&product); err != nil {
// 			skipped++
// 			continue
// 		}

// 		if product.Productphotourl != "" && !strings.Contains(product.Productphotourl, cdnDomain) {
// 			fullPath := filepath.Join("./uploads/products", product.Productphotourl)

// 			fileData, err := os.Open(fullPath)
// 			if err != nil {
// 				skipped++
// 				continue
// 			}
// 			defer fileData.Close()

// 			fileInfo, err := fileData.Stat()
// 			if err != nil {
// 				skipped++
// 				continue
// 			}

// 			ext := filepath.Ext(fileInfo.Name())
// 			if ext == "" {
// 				ext = ".jpg"
// 			}

// 			baseFilename := fmt.Sprintf("products/%s_migrated", product.ID.Hex())
// 			mainFilename := baseFilename + ext
// 			previewFilename := baseFilename + "_preview" + ext

// 			_, err = s3Client.PutObject(context.Background(), s3Bucket, mainFilename, fileData, fileInfo.Size(), minio.PutObjectOptions{
// 				ContentType: "image/jpeg",
// 			})
// 			if err != nil {
// 				skipped++
// 				continue
// 			}

// 			fileData.Seek(0, 0)
// 			previewBuf := &bytes.Buffer{}
// 			img, _, err := image.Decode(fileData)
// 			if err == nil {
// 				previewImg := resize.Thumbnail(previewSize, previewSize, img, resize.Lanczos3)
// 				jpeg.Encode(previewBuf, previewImg, &jpeg.Options{Quality: 75})
// 				s3Client.PutObject(context.Background(), s3Bucket, previewFilename, previewBuf, int64(previewBuf.Len()), minio.PutObjectOptions{
// 					ContentType: "image/jpeg",
// 				})
// 			}

// 			mainURL := fmt.Sprintf("https://%s/%s", cdnDomain, mainFilename)
// 			previewURL := fmt.Sprintf("https://%s/%s", cdnDomain, previewFilename)
// 			update := bson.M{
// 				"$set": bson.M{
// 					"productphotourl":        mainURL,
// 					"productphotopreviewurl": previewURL,
// 				},
// 				"$unset": bson.M{
// 					"photourl":        "",
// 					"photopreviewurl": "",
// 				},
// 			}
// 			_, err = config.ProductTemplateCollection.UpdateByID(context.TODO(), product.ID, update)
// 			if err != nil {
// 				skipped++
// 				continue
// 			}

// 			migrated++
// 		}
// 	}

// 	cursor, err = config.ProductCollection.Find(context.TODO(), bson.M{})
// 	if err == nil {
// 		defer cursor.Close(context.TODO())
// 		for cursor.Next(context.TODO()) {
// 			var prod models.Product
// 			if err := cursor.Decode(&prod); err != nil {
// 				skipped++
// 				continue
// 			}
// 			if prod.Productphotourl != "" && !strings.Contains(prod.Productphotourl, cdnDomain) {
// 				fullPath := filepath.Join("./uploads/products", prod.Productphotourl)
// 				fileData, err := os.Open(fullPath)
// 				if err != nil {
// 					skipped++
// 					continue
// 				}
// 				defer fileData.Close()
// 				fileInfo, err := fileData.Stat()
// 				if err != nil {
// 					skipped++
// 					continue
// 				}
// 				ext := filepath.Ext(fileInfo.Name())
// 				if ext == "" {
// 					ext = ".jpg"
// 				}
// 				baseFilename := fmt.Sprintf("products/%s_migrated", prod.ID.Hex())
// 				mainFilename := baseFilename + ext
// 				previewFilename := baseFilename + "_preview" + ext

// 				_, err = s3Client.PutObject(context.Background(), s3Bucket, mainFilename, fileData, fileInfo.Size(), minio.PutObjectOptions{
// 					ContentType: "image/jpeg",
// 				})
// 				if err != nil {
// 					skipped++
// 					continue
// 				}

// 				fileData.Seek(0, 0)
// 				previewBuf := &bytes.Buffer{}
// 				img, _, err := image.Decode(fileData)
// 				if err == nil {
// 					previewImg := resize.Thumbnail(previewSize, previewSize, img, resize.Lanczos3)
// 					jpeg.Encode(previewBuf, previewImg, &jpeg.Options{Quality: 75})
// 					s3Client.PutObject(context.Background(), s3Bucket, previewFilename, previewBuf, int64(previewBuf.Len()), minio.PutObjectOptions{
// 						ContentType: "image/jpeg",
// 					})
// 				}

// 				mainURL := fmt.Sprintf("https://%s/%s", cdnDomain, mainFilename)
// 				previewURL := fmt.Sprintf("https://%s/%s", cdnDomain, previewFilename)
// 				update := bson.M{
// 					"$set": bson.M{
// 						"productphotourl":        mainURL,
// 						"productphotopreviewurl": previewURL,
// 					},
// 					"$unset": bson.M{
// 						"photourl":        "",
// 						"photopreviewurl": "",
// 					},
// 				}
// 				_, err = config.ProductCollection.UpdateByID(context.TODO(), prod.ID, update)
// 				if err != nil {
// 					skipped++
// 					continue
// 				}
// 				migrated++
// 			}
// 		}
// 	}

// 	log.Printf("[Migration] migrated: %d | skipped: %d\n", migrated, skipped)
// }
// // MigrateCategoryImages переносит локальные фото категорий в S3
// func MigrateCategoryImages() {
// 	cursor, err := config.CategoryCollection.Find(context.TODO(), bson.M{})
// 	if err != nil {
// 		log.Println("[Migration] Failed to fetch categories:", err)
// 		return
// 	}
// 	defer cursor.Close(context.TODO())

// 	migrated := 0
// 	skipped := 0

// 	for cursor.Next(context.TODO()) {
// 		var cat models.Category
// 		if err := cursor.Decode(&cat); err != nil {
// 			skipped++
// 			continue
// 		}
// 		if cat.PhotoURL != "" && !strings.Contains(cat.PhotoURL, cdnDomain) {
// 			fullPath := filepath.Join("./uploads/categories", cat.PhotoURL)
// 			fileData, err := os.Open(fullPath)
// 			if err != nil {
// 				skipped++
// 				continue
// 			}
// 			defer fileData.Close()
// 			fileInfo, err := fileData.Stat()
// 			if err != nil {
// 				skipped++
// 				continue
// 			}
// 			ext := filepath.Ext(fullPath)
// 			if ext == "" {
// 				ext = ".jpg"
// 			}
// 			mainFilename := fmt.Sprintf("categories/%s_migrated%s", cat.ID.Hex(), ext)
// 			_, err = s3Client.PutObject(context.Background(), s3Bucket, mainFilename, fileData, fileInfo.Size(), minio.PutObjectOptions{
// 				ContentType: "image/jpeg",
// 			})
// 			if err != nil {
// 				skipped++
// 				continue
// 			}
// 			mainURL := fmt.Sprintf("https://%s/%s", cdnDomain, mainFilename)
// 			update := bson.M{"$set": bson.M{"photourl": mainURL}}
// 			_, err = config.CategoryCollection.UpdateByID(context.TODO(), cat.ID, update)
// 			if err != nil {
// 				skipped++
// 				continue
// 			}
// 			migrated++
// 		}
// 	}
// 	log.Printf("[Category Migration] migrated: %d | skipped: %d\n", migrated, skipped)
// }



