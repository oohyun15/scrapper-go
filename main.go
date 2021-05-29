package main

import (
	"os"
	"strings"

	"github.com/oohyun15/scrapper-go/scrapper"

	"github.com/labstack/echo"
)

const fileName string = "webtoon.csv"

func handleHome(c echo.Context) error {
	return c.File("home.html")
}

func handleScrape(c echo.Context) error {
	defer os.Remove(fileName)
	term := strings.ToLower(scrapper.CleanString(c.FormValue("term")))
	pivot := strings.ToLower(scrapper.CleanString(c.FormValue("pivot")))
	scrapper.Scrape(term, pivot)
	// return nil
	return c.Attachment(fileName, pivot+"_"+term+"_"+fileName)
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
