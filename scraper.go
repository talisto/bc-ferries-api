package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Route struct {
	SailingDuration string    `json:"sailingDuration"`
	Sailings        []Sailing `json:"sailings"`
}

type Sailing struct {
	DepartureDate string `json:"date"`
	DepartureTime string `json:"time"`
	ArrivalTime   string `json:"arrivalTime"`
	IsCancelled   bool   `json:"isCancelled"`
	Fill          int    `json:"fill"`
	VesselName    string `json:"vesselName"`
}

func MakeCurrentConditionsLink(departure, destination string) string {
	return "https://www.bcferries.com/current-conditions/" + departure + "-" + destination
}

func MakeScheduleLink(departure, destination string) string {
	return "https://www.bcferries.com/routes-fares/schedules/daily/" + departure + "-" + destination
}

func ContainsSailingData(stringToCheck string) bool {
	if strings.Contains(stringToCheck, "%") || strings.Contains(stringToCheck, "FULL") || strings.Contains(stringToCheck, "Cancelled") || strings.Contains(stringToCheck, "Full") {
		return true
	}

	return false
}

func ScrapeRoutes(localMode bool) map[string]map[string]Route {
	departureTerminals := GetDepartureTerminals()

	destinationTerminals := GetDestinationTerminals()

	var schedule = make(map[string]map[string]Route)

	for i := 0; i < len(departureTerminals); i++ {
		schedule[departureTerminals[i]] = make(map[string]Route)

		for j := 0; j < len(destinationTerminals[i]); j++ {
			var document *goquery.Document

			if localMode {
				file, err := os.OpenFile("./sample/sample-site.html", os.O_RDWR, 0644)
				if err != nil {
					log.Fatal("Local file read failed")
				}

				var response io.Reader = (file)

				// Create a goquery document from the HTTP response
				document, err = goquery.NewDocumentFromReader(response)
				if err != nil {
					log.Fatal("Error loading HTTP response body. ", err)
				}
			} else {
				link := ""
				if departureTerminals[i] == "FUL" || departureTerminals[i] == "BOW" {
					link = MakeScheduleLink(departureTerminals[i], destinationTerminals[i][j])
				} else {
					link = MakeCurrentConditionsLink(departureTerminals[i], destinationTerminals[i][j])
				}

				// Make HTTP GET request
				client := &http.Client{}
				req, _ := http.NewRequest("GET", link, nil)
				req.Header.Add("User-Agent", "Mozilla")
				response, err := client.Do(req)

				if err != nil {
					log.Fatal(err)
				}

				defer response.Body.Close()

				document, err = goquery.NewDocumentFromReader(response.Body)
				if err != nil {
					log.Fatal("Error loading HTTP response body. ", err)
				}
			}

			if departureTerminals[i] == "FUL" || departureTerminals[i] == "BOW" {
				route := ScrapeNonCapacityRoute(document)

				schedule[departureTerminals[i]][destinationTerminals[i][j]] = route
			} else {
				route := ScrapeCapacityRoute(document)

				schedule[departureTerminals[i]][destinationTerminals[i][j]] = route
			}
		}
	}

	return schedule
}

func ScrapeCapacityRoute(document *goquery.Document) Route {
	route := Route{
		SailingDuration: "",
		Sailings:        []Sailing{},
	}

	loc, _ := time.LoadLocation("America/Vancouver")
	//set timezone,
	now := time.Now().In(loc)

	// Get table of times and capacities
	document.Find(".mobile-friendly-row").Each(func(index int, sailingData *goquery.Selection) {

		sailing := Sailing{}

		if ContainsSailingData(sailingData.Text()) {
			// TIME AND VESSEL NAME
			timeAndBoatName := sailingData.Find(".mobile-paragraph").First().Text()
			timeAndBoatNameArray := strings.Split(timeAndBoatName, "\n")

			if strings.Contains(timeAndBoatName, "Tomorrow") {
				sailing.DepartureDate = now.AddDate(0, 0, 1).Format("2006-01-02")
			} else if strings.Contains(timeAndBoatName, now.AddDate(0, 0, 2).Format("Jan 02, 2006")) {
				sailing.DepartureDate = now.AddDate(0, 0, 2).Format("2006-01-02")
			} else {
				sailing.DepartureDate = now.Format("2006-01-02")
			}

			for i := 0; i < len(timeAndBoatNameArray); i++ {
				item := strings.TrimSpace(timeAndBoatNameArray[i])
				item = strings.ReplaceAll(item, "\n", "")

				re := regexp.MustCompile(`(1?[0-9]:[0-5][0-9] (am|pm|AM|PM))`)
				time := re.FindString(item)

				if time != "" {
					sailing.DepartureTime = time
				}
			}

			sailing.VesselName = strings.TrimSpace(sailingData.Find(".sailing-ferry-name").First().Text())

			// FILL
			fill := strings.TrimSpace(sailingData.Find(".cc-vessel-percent-full").First().Text())
			if fill == "FULL" || fill == "Full" {
				sailing.Fill = 100
				sailing.IsCancelled = false
			} else if strings.Contains(fill, "Cancelled") {
				sailing.Fill = 0
				sailing.IsCancelled = true
			} else {
				fill, err := strconv.Atoi(strings.Split(fill, "%")[0])
				if err == nil {
					sailing.Fill = 100 - fill
				}
				sailing.IsCancelled = false
			}

			route.Sailings = append(route.Sailings, sailing)
		}
	})

	return route
}

func ScrapeNonCapacityRoute(document *goquery.Document) Route {
	route := Route{
		SailingDuration: "",
		Sailings:        []Sailing{},
	}

	loc, _ := time.LoadLocation("America/Vancouver")
	//set timezone,
	now := time.Now().In(loc)

	re := regexp.MustCompile(`(1?[0-9]:[0-5][0-9] (am|pm|AM|PM))`)

	document.Find("#seasonalSchedulesForm .table-seasonal-schedule tbody").First().Find(".schedule-table-row").Each(func(index int, sailingData *goquery.Selection) {
		sailing := Sailing{}

		sailing.DepartureDate = now.Format("2006-01-02")

		sailingData.Find("td").Each(func(index int, sailingData *goquery.Selection) {
			if index == 1 {
				sailing.DepartureTime = re.FindString(strings.TrimSpace(sailingData.Text()))
			} else if index == 2 {
				sailing.ArrivalTime = re.FindString(strings.TrimSpace(sailingData.Text()))
			}
		})

		route.Sailings = append(route.Sailings, sailing)
	})

	return route
}
