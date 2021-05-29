package main

import (
	"os"
	"strconv"

	"github.com/oohyun15/scrapper-go/scrapper"

	"github.com/labstack/echo"
)

const fileName string = "webtoon.csv"

func handleHome(c echo.Context) error {
	return c.File("home.html")
}

func handleScrape(c echo.Context) error {
	defer os.Remove(fileName)
	start, _ := strconv.Atoi(scrapper.CleanString(c.FormValue("start")))
	end, _ := strconv.Atoi(scrapper.CleanString(c.FormValue("end")))
	batchSize, _ := strconv.Atoi(scrapper.CleanString(c.FormValue("batch")))
	scrapper.Scrape(start, end, batchSize)
	return c.Attachment(fileName, "webtoon("+scrapper.CleanString(c.FormValue("start"))+"-"+scrapper.CleanString(c.FormValue("end"))+").csv")
}

func Rescrape(c echo.Context) error {
	defer os.Remove(fileName)
	scrapper.Rescrape()
	// return nil
	return c.Attachment(fileName, "rescrape_"+fileName)
}

func main() {
	e := echo.New()
	e.GET("/", handleHome)
	e.POST("/scrape", handleScrape)
	e.POST("/rescrape", Rescrape)
	e.Logger.Fatal(e.Start(":3000"))
}
