package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// XML ...
type XML struct {
	Channel struct {
		Items []struct {
			Title     string    `xml:"title"`
			Enclosure Enclosure `xml:"enclosure"`
		} `xml:"item"`
	} `xml:"channel"`
}

// Enclosure ...
type Enclosure struct {
	Name string `xml:",chardata"`
	URL  string `xml:"url,attr"`
}

// JSON ...
type JSON struct {
	Items []Item `json:"items"`
}

// Item ...
type Item struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func main() {
	data := httpGet("http://feeds.rebuild.fm/rebuildfm")

	result := XML{}
	err := xml.Unmarshal([]byte(data), &result)
	if err != nil {
		log.Fatalf("error: %v", err)
		return
	}
	jsonObj := new(JSON)
	for index, item := range result.Channel.Items {
		i := new(Item)
		i.Title = item.Title
		i.URL = strings.Replace(item.Enclosure.URL, "http://cache.rebuild.fm/", "https://s3-ap-northeast-1.amazonaws.com/rebuild.bucket/mp3/", -1)
		jsonObj.Items = append(jsonObj.Items, *i)
		if index == 0 {
			tokens := strings.Split(item.Enclosure.URL, "/")
			fileName := tokens[len(tokens)-1]
			if checkExist("mp3/", fileName) {
				fmt.Println(fileName + " exist")
			} else {
				fmt.Println(fileName + " not exist")
				filename, err := downloadFromURL(item.Enclosure.URL)
				if err != nil {
					log.Fatalf("error: %v", err)
					return
				}
				fileURL, err := PutToS3(filename, "/mp3/", "audio/mpeg")
				if err != nil {
					log.Fatalf("error: %v", err)
					return
				}
				fmt.Println(fileURL)
			}
		}
	}

	b, err := json.Marshal(jsonObj)
	if err != nil {
		log.Fatalf("error: %v", err)
		return
	}
	err = ioutil.WriteFile("/tmp/rebuild.json", b, os.ModePerm)
	if err != nil {
		log.Fatalf("error: %v", err)
		return
	}

	fileURL, err := PutToS3("rebuild.json", "/", "application/json")
	if err != nil {
		log.Fatalf("error: %v", err)
		return
	}
	fmt.Println("Uploaded: " + fileURL)
}

func httpGet(url string) string {
	response, _ := http.Get(url)
	body, _ := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	return string(body)
}

func downloadFromURL(url string) (string, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fmt.Println("Downloading", url, "to", fileName)

	output, err := os.Create("/tmp/" + fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return "", err
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return "", err
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return "", err
	}

	fmt.Println(n, "bytes downloaded.")
	return fileName, nil
}

//********************** S3 ****************//

const s3Path = "https://s3-ap-northeast-1.amazonaws.com/rebuild.bucket/"

// PutToS3 ...
func PutToS3(fileName string, path string, contentType string) (string, error) {
	file, err := os.Open("/tmp/" + fileName)
	if err != nil {
		return "", err
	}
	cli := getS3Cli()

	_, err = cli.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String("rebuild.bucket"),
		Key:         aws.String(path + fileName),
		Body:        file,
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		return "", err
	}

	return s3Path + fileName, nil
}

func checkExist(path, fileName string) bool {
	cli := getS3Cli()
	fileList, err := cli.ListObjects(&s3.ListObjectsInput{
		Bucket:  aws.String("rebuild.bucket"),
		Prefix:  aws.String(path + fileName),
		MaxKeys: aws.Int64(1),
	})
	if err != nil {
		return false
	}
	if len(fileList.Contents) == 0 {
		return false
	}
	return true
}

func getS3Cli() *s3.S3 {
	cre := credentials.NewStaticCredentials(
		os.Getenv("AccessKeyID"),
		os.Getenv("SecretAccessKey"),
		"")

	cli := s3.New(session.New(), &aws.Config{
		Credentials: cre,
		Region:      aws.String("ap-northeast-1"),
	})
	return cli
}
