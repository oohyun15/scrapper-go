package scrapper

import (
	"encoding/csv"
	"fmt"
	"io"
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
	author_id        string
	publisher        string
	origin_publisher string
	description      string
	price            string
	size             string
	isbn             string
	published_at     string
	image            string
	date             string
	link             string
}

type tableData struct {
	title string
	name  string
}

var Num int

func Rescrape() {
	ids := readIds()
	fmt.Println("count:", len(ids))
	startTime := time.Now()
	fmt.Println("start:", startTime)
	w := initFile()
	var jobs []extractedJob
	var baseURL string = "http://dml.komacon.kr/archive/"
	c := make(chan extractedJob)
	count := len(ids)
	batchSize := 100

	for idx := 0; idx < count/batchSize+1; idx++ {
		start := idx * batchSize
		end := (idx + 1) * batchSize
		if end > count {
			end = count
		}
		fmt.Println("start:", start, "end:", end)
		for i := start; i < end; i++ {
			num, _ := strconv.Atoi(ids[i])
			go getPage(num, baseURL, c)
		}
		for i := start; i < end; i++ {
			extractedJobs := <-c
			jobs = append(jobs, extractedJobs)
		}
	}

	writeJobs(jobs, w)
	fmt.Println("Done, extracted")
	endTime := time.Now()
	fmt.Println("end: ", endTime)
}

// Scrape indeeds term
func Scrape(start int, end int, batchSize int) {
	startTime := time.Now()
	fmt.Println("start:", startTime)
	w := initFile()
	Num = 0

	var jobs []extractedJob
	var baseURL string = "http://dml.komacon.kr/archive/"
	c := make(chan extractedJob)

	for idx := 0; idx < (end-start)/batchSize+1; idx++ {
		_start := idx*batchSize + start
		_end := (idx+1)*batchSize + start
		if _end > end {
			_end = end
		}
		fmt.Println("start:", _start, "end:", _end-1, "current:", Num)
		for i := _start; i < _end; i++ {
			go getPage(i, baseURL, c)
		}
		for i := _start; i < _end; i++ {
			extractedJobs := <-c
			jobs = append(jobs, extractedJobs)
		}
	}
	writeJobs(jobs, w)
	fmt.Println("Done, extracted", Num)
	endTime := time.Now()
	fmt.Println("end: ", endTime)
}

func getPage(id int, url string, mainC chan<- extractedJob) {
	var job extractedJob
	job.id = id
	pageURL := url + strconv.Itoa(id)
	// fmt.Println("pageURL:", pageURL)
	var res *http.Response
	var err error
	if res == nil {
		res, err = http.Get(pageURL)
	}

	checkErr(err)
	checkCode(res, pageURL)
	if res == nil {
		fmt.Println("Response is nil", pageURL)
		mainC <- job
		return
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)
	titleList := strings.Split(doc.Find(".arcive-base-data").Text(), "\n")
	if len(titleList) == 1 {
		mainC <- job
		// fmt.Println("Not found", pageURL)
		return
	}
	title := findTitle(titleList, id)
	job.title = CleanString(title)
	// job.description = strings.TrimSpace(doc.Find(".content").Text())
	image, _ := doc.Find(".arcive-img").Attr("style")
	image = strings.Split(image, ",")[0]
	image = strings.Split(image, "background-image: url('")[1]
	image = strings.Split(image, "')")[0]
	job.image = image

	author_id, _ := doc.Find("td a").Attr("href")
	if author_id != "" && strings.Contains(author_id, "author") {
		job.author_id = strings.Split(author_id, "/author/")[1]
	}

	dataTable := doc.Find(".arcive-data-table tr")
	dataTable.Each(func(i int, card *goquery.Selection) {
		title := CleanString(card.Find("td.td-header").Text())
		name := CleanString(card.Find("td").Last().Text())
		data := tableData{
			title: title,
			name:  name,
		}
		webtoonTitle(data, &job)
	})

	link, _ := doc.Find("a.btn").Attr("href")
	job.link = link
	mainC <- job
}

func extractJob(card *goquery.Selection, c chan<- tableData) {
	title := CleanString(card.Find("td.td-header").Text())
	name := CleanString(card.Find("td").Last().Text())
	c <- tableData{
		title: title,
		name:  name,
	}
}

// CleanString cleans string
func CleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}

func initFile() *csv.Writer {
	file, err := os.Create("webtoon.csv")
	checkErr(err)

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"id", "title", "type", "author", "author_id", "publisher", "image", "date", "link"}

	wErr := w.Write(headers)
	checkErr(wErr)

	return w
}

func writeJobs(jobs []extractedJob, w *csv.Writer) {
	defer w.Flush()
	c := make(chan error)
	for _, job := range jobs {
		if job._type == "웹툰" {
			go writeJob(job, w, c)
			checkErr(<-c)
		}
	}
}

func writeJob(job extractedJob, w *csv.Writer, writeC chan<- error) {
	jobSlice := []string{
		strconv.Itoa(job.id),
		job.title,
		job._type,
		job.author,
		job.author_id,
		job.publisher,
		// job.description,
		job.image,
		job.date,
		job.link,
	}
	jwErr := w.Write(jobSlice)
	writeC <- jwErr
}

func checkErr(err error) {
	if err := recover(); err != nil {
		panic(err)
	}
}

func checkCode(res *http.Response, url string) {
	defer func() {
		if c := recover(); c != nil {
			fmt.Println("recover", url)
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

func webtoonTitle(data tableData, job *extractedJob) {
	switch data.title {
	case "형태":
		job._type = data.name
		if job._type == "웹툰" {
			Num += 1
		}
	case "작가":
		job.author = data.name
	case "연재매체":
		job.publisher = data.name
	case "출판사":
		job.publisher = data.name
	case "연재기간":
		job.date = data.name
	}
}

func readIds() []string {
	var ids []string
	file, err := os.Open("webtoons.csv")
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
	}
	r := csv.NewReader(file)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
		}
		ids = append(ids, record[0])
	}

	fmt.Println("ids.count", len(ids))

	keys := make(map[string]bool)
	ue := []string{}

	for _, value := range ids {
		if _, saveValue := keys[value]; !saveValue {

			keys[value] = true
			ue = append(ue, value)
		} else {
			fmt.Println(value)
		}
	}
	fmt.Println("ue.count", len(ue))
	return ue
}
