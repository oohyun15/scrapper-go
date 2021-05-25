package scrapper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type extractedJob struct {
	id               int
	_type            string
	title            string
	author           string
	publisher        string
	origin_publisher string
	description      string
	price            string
	size             string
	isbn             string
	published_at     string

	// "id", "type", "author", "publisher", "origin_publisher", "price", "size", "isbn", "published_at"
	// title    string
	// location string
	// salary   string
	// summary  string
}

type tableData struct {
	title string
	name  string
}

// Scrape indeeds term
func Scrape(term string) {
	startTime := time.Now()
	fmt.Println("start:", startTime)
	var jobs []extractedJob
	var baseURL string = "http://dml.komacon.kr/archive/"
	c := make(chan extractedJob)
	count, _ := strconv.Atoi(term)
	batchSize := 500

	for idx := 1; idx < (count-150000)/batchSize+1; idx++ {
		start := idx*batchSize + 150000
		end := (idx+1)*batchSize + 150000
		if end > count {
			end = count
		}
		fmt.Println("start:", start, "end:", end)
		for i := start; i < end; i++ {
			go getPage(i, baseURL, c)
		}
		for i := start; i < end; i++ {
			extractedJobs := <-c
			// fmt.Println(extractedJobs)
			jobs = append(jobs, extractedJobs)
		}
	}
	writeJobs(jobs)
	fmt.Println("Done, extracted", len(jobs))
	endTime := time.Now()
	fmt.Println("end: ", endTime)
}

func getPage(id int, url string, mainC chan<- extractedJob) {
	var job extractedJob
	job.id = id
	c := make(chan tableData)
	pageURL := url + strconv.Itoa(id)
	// fmt.Println("Requesting", pageURL)
	res, err := http.Get(pageURL)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)
	titleList := strings.Split(doc.Find(".arcive-base-data").Text(), "\n")
	if len(titleList) == 1 {
		mainC <- job
		return
	}
	title := findTitle(titleList, id)
	job.title = CleanString(title)
	job.description = strings.TrimSpace(doc.Find(".content").Text())
	dataTable := doc.Find(".arcive-data-table tr")
	dataTable.Each(func(i int, card *goquery.Selection) {
		go extractJob(card, c)
	})
	for i := 0; i < dataTable.Length(); i++ {
		data := <-c
		convertTitle(data, &job)
	}
	mainC <- job
}

func extractJob(card *goquery.Selection, c chan<- tableData) {
	title := CleanString(card.Find("td.td-header").Text())
	name := CleanString(card.Find("td").Last().Text())

	// id, _ := card.Attr("data-jk")
	// title := CleanString(card.Find(".title>a").Text())
	// location := CleanString(card.Find(".sjcl").Text())
	// salary := CleanString(card.Find(".salaryText").Text())
	// summary := CleanString(card.Find(".summary").Text())
	c <- tableData{
		title: title,
		name:  name,
	}
}

// CleanString cleans string
func CleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}

func getPages(url string) int {
	pages := 0
	res, err := http.Get(url)
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	doc.Find(".pagination").Each(func(i int, s *goquery.Selection) {
		pages = s.Find("a").Length()
	})

	return pages
}

func writeJobs(jobs []extractedJob) {
	file, err := os.Create("webtoon.csv")
	checkErr(err)

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"id", "title", "type", "author", "publisher", "origin_publisher", "description", "price", "size", "isbn", "published_at"}

	wErr := w.Write(headers)
	checkErr(wErr)

	c := make(chan error)

	for _, job := range jobs {
		go writeJob(job, w, c)

		checkErr(<-c)
	}
}

func writeJob(job extractedJob, w *csv.Writer, writeC chan<- error) {
	jobSlice := []string{
		strconv.Itoa(job.id),
		job.title,
		job._type,
		job.author,
		job.publisher,
		job.origin_publisher,
		job.description,
		job.price,
		job.size,
		job.isbn,
		job.published_at,
	}
	jwErr := w.Write(jobSlice)
	writeC <- jwErr
}

func checkErr(err error) {
	if err := recover(); err != nil {
		fmt.Println(err)
	}
}

func checkCode(res *http.Response) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("panic occurred:", err)
		}
	}()

	if res.StatusCode != 200 {
		log.Fatalln("Request failed with Status:", res.StatusCode)
	}
}

func findTitle(title []string, id int) string {
	defer func() {
		if c := recover(); c != nil {
			fmt.Println("recover", id)
		}
	}()
	return title[1]
}

func convertTitle(data tableData, job *extractedJob) {
	switch data.title {
	case "형태":
		job._type = data.name
	case "작가":
		job.author = data.name
	case "출판사":
		job.publisher = data.name
	case "원출판사":
		job.origin_publisher = data.name
	case "정가":
		val := strings.Split(data.name, "원")[0]
		val = strings.Replace(val, ",", "", -1)
		job.price = val
	case "판형/페이지":
		job.size = data.name
	case "ISBN":
		job.isbn = data.name
	case "출판일":
		job.published_at = data.name
	}
}
