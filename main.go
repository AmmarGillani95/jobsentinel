package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/smithy-go"
)

// 24 hours 604800
// 5 mins 300
const (
	baseURL          = "https://www.linkedin.com/jobs/search/?"
	keywords         = "Engineer OR Developer AND (Golang OR Typescript OR React)"
	geoId            = "103644278" // USA
	remote           = "2"         // remote
	secondsSincePost = 600         // 10 minutes
	s3Bucket         = "slinkedin-jobs"
	s3Key            = "sent_jobs.json"
	location         = "San Antonio"
	distance         = "100"
)

type Event struct {
	Email string `json:"email"`
}

// Job represents a LinkedIn job post
type Job struct {
	Id       string
	Title    string
	Company  string
	Location string
	URL      string
	PostedAt string
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_3_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.5993.88 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:114.0) Gecko/20100101 Firefox/114.0",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:109.0) Gecko/20100101 Firefox/109.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.6045.134 Mobile Safari/537.36",
	"Mozilla/5.0 (iPad; CPU OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 11_7_10) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.4 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:118.0) Gecko/20100101 Firefox/118.0",
	"Mozilla/5.0 (Linux; Android 12; SM-G991U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.5993.65 Mobile Safari/537.36",
}

var referers = []string{
	"https://www.google.com/",
	"https://www.bing.com/",
	"https://news.ycombinator.com/",
}

var languages = []string{
	"en-US,en;q=0.9",
	"en-GB,en;q=0.8",
	"en-AU,en;q=0.9",
	"en-CA,en;q=0.9",
}
var blockedPosters = map[string]string{
	"DataAnnotation": "true",
	"Jobs via Dice":  "true",
}

func getRandomReferer() string {
	return referers[rand.Intn(len(referers))]
}

func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}
func getRandomLanguage() string {
	return languages[rand.Intn(len(languages))]
}

func buildURL(locationType string) *url.URL {
	u, _ := url.Parse(baseURL)
	params := url.Values{}
	if locationType == "remote" {
		params.Set("geoId", geoId)
		params.Set("f_WT", remote)
		params.Set("keywords", keywords)
		params.Set("f_TPR", "r"+strconv.Itoa(secondsSincePost))
	} else {
		params.Set("keywords", keywords)
		params.Set("f_TPR", "r"+strconv.Itoa(secondsSincePost))
		params.Set("location", location)
		params.Set("distance", distance)
	}
	u.RawQuery = params.Encode()
	return u
}

func loadSentURNs(ctx context.Context, s3Client *s3.Client, bucket, key string) (map[string]bool, error) {
	result := make(map[string]bool)

	resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check for NoSuchKey error properly
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
			// File doesn't exist yet — return empty result
			return result, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func saveSentURNs(ctx context.Context, s3Client *s3.Client, bucket, key string, urns map[string]bool) error {
	data, err := json.Marshal(urns)
	if err != nil {
		return err
	}

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader(string(data)),
	})
	return err
}

// Mock function - replace with actual LinkedIn API logic
func FetchLinkedInJobs(locationType string) (jobs []Job, err error) {
	url := buildURL(locationType)
	fmt.Printf("Checking jobs on %s\n", url)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", getRandomLanguage())
	req.Header.Set("Referer", getRandomReferer())
	req.Header.Set("DNT", "1") // Do Not Track
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Error loading HTML:", err)
		return
	}
	//#main-content // main div job results?
	//.no-results class for zero results
	if doc.Find(".no-results").Length() > 0 {
		return jobs, fmt.Errorf("no results")
	}
	doc.Find(".jobs-search__results-list li").Each(func(i int, s *goquery.Selection) {
		var job Job
		urn, _ := s.Find(".base-card").Attr("data-entity-urn")
		job.Id = strings.TrimPrefix(urn, "urn:li:jobPosting:")
		job.Title = strings.TrimSpace(s.Find(".base-search-card__title").Text())
		job.Company = strings.TrimSpace(s.Find(".base-search-card__subtitle a").Text())
		job.Location = strings.TrimSpace(s.Find(".job-search-card__location").Text())
		job.URL, _ = s.Find(".base-card__full-link").Attr("href")
		job.PostedAt = strings.TrimSpace(s.Find(".job-search-card__listdate").Text())
		jobs = append(jobs, job)
	})

	return jobs, nil
}

// Sends email with job list
func SendEmail(jobs []Job, sesClient *ses.Client, subject string, ctx context.Context, email string) error {
	var body string
	for _, job := range jobs {
		body += fmt.Sprintf("Title: %s\nCompany: %s\nLocation: %s\nLink: %s\n\n", job.Title, job.Company, job.Location, job.URL)
	}

	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{email},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: aws.String(subject),
			},
			Body: &types.Body{
				Text: &types.Content{
					Data: aws.String(body),
				},
			},
		},
		Source: aws.String(email),
	}

	_, err := sesClient.SendEmail(ctx, input)
	return err
}

func filterResults(jobs []Job) []Job {
	var newJobs []Job
	for _, job := range jobs {
		if _, found := blockedPosters[job.Company]; !found {
			newJobs = append(newJobs, job)
		} else {
			fmt.Printf("Filtered out job from poster: %s\n", job.Company)
		}
	}
	return newJobs
}

func handler(ctx context.Context, event Event) error {
	if event.Email == "" {
		return fmt.Errorf("email parameter is required")
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	s3Client := s3.NewFromConfig(cfg)
	sentURNs, err := loadSentURNs(ctx, s3Client, s3Bucket, s3Key)
	if err != nil {
		return fmt.Errorf("failed to load sent URNs: %v", err)
	}

	remoteJobs, err := FetchLinkedInJobs("remote")
	if err != nil {
		return fmt.Errorf("failed to fetch jobs: %v", err)
	}

	var newRemoteJobs []Job
	for _, job := range remoteJobs {
		if !sentURNs[job.Id] {
			newRemoteJobs = append(newRemoteJobs, job)
			sentURNs[job.Id] = true // Mark for saving
		}
	}

	filteredRemoteJobs := filterResults((newRemoteJobs))

	if len(filteredRemoteJobs) == 0 {
		fmt.Println("No new remote jobs to send.")
		return nil
	}

	sesClient := ses.NewFromConfig(cfg)

	subject := fmt.Sprintf("New Remote LinkedIn Jobs Found! (%d jobs)", len(filteredRemoteJobs))
	err = SendEmail(filteredRemoteJobs, sesClient, subject, ctx, event.Email)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}
	fmt.Printf("Sent %d new remote jobs via email.\n", len(filteredRemoteJobs))

	localJobs, err := FetchLinkedInJobs("local")
	if err != nil {
		return fmt.Errorf("failed to fetch local jobs: %v", err)
	}

	var newLocalJobs []Job
	for _, job := range localJobs {
		if !sentURNs[job.Id] {
			newLocalJobs = append(newLocalJobs, job)
			sentURNs[job.Id] = true // Mark for saving
		}
	}

	filteredLocalJobs := filterResults((newLocalJobs))
	if len(filteredLocalJobs) == 0 {
		fmt.Println("No new local jobs to send.")
		return nil
	}
	subject = fmt.Sprintf("New Local LinkedIn Jobs Found! (%d jobs)", len(filteredLocalJobs))
	err = SendEmail(filteredLocalJobs, sesClient, subject, ctx, event.Email)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}
	fmt.Printf("Sent %d new local jobs via email.\n", len(filteredLocalJobs))
	err = saveSentURNs(ctx, s3Client, s3Bucket, s3Key, sentURNs)
	if err != nil {
		return fmt.Errorf("failed to save sent URNs: %v", err)
	}
	return nil
}

func main() {
	//handler(context.Background(), Event{Email: "jonathanjface"})
	lambda.Start(handler)
}
