package cloudscrap

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gocolly/colly/v2"
	"github.com/gregdel/pushover"
)

var (
	asset   = os.Getenv("ASSET")
	h       = os.Getenv("HIGH")
	l       = os.Getenv("LOW")
	appID   = os.Getenv("APP_ID")
	groupID = os.Getenv("GROUP_ID")
)

func init() {}

func main() {}

/*CloudAlert - ..
 */
func CloudAlert(w http.ResponseWriter, r *http.Request) {
	var high, low float64
	query := r.URL.Query()
	if hq := query.Get("h"); hq != "" {

		high, _ = strconv.ParseFloat(hq, 64)
	} else {
		high, _ = strconv.ParseFloat(h, 64)
	}
	if lq := query.Get("l"); lq != "" {
		low, _ = strconv.ParseFloat(lq, 64)
	} else {
		low, _ = strconv.ParseFloat(l, 64)
	}
	if a := query.Get("a"); a != "" {
		asset = a
	}
	bid := scrap(&high, &low)
	w.WriteHeader(http.StatusOK)
	response := bid
	if err := json.NewEncoder(w).Encode(&response); err != nil {
		log.Println(err.Error())
	}
}

func scrap(high *float64, low *float64) string {
	var b = "OK"
	// Instantiate default collector
	c := colly.NewCollector()
	c.OnHTML("div[id=symbol-bid]", func(e *colly.HTMLElement) {
		b = e.Text
		if bid, err := strconv.ParseFloat(b, 64); err == nil {
			log.Printf("High: %.2f, Low: %.2f", *high, *low)
			if bid > *high {
				msg := fmt.Sprintf("%s is now at %.2f", asset, bid)
				alert(msg, "Making money")
			} else if bid < *low {
				msg := fmt.Sprintf("%s is now at %.2f", asset, bid)
				alert(msg, "Losing money!")
			} else {
				log.Printf("Bid: %.2f", bid)
			}
		}
	})
	url := fmt.Sprintf("https://bossafx.pl/oferta/instrumenty/%s.", asset)
	c.Visit(url)
	// Wait until threads are finished
	// c.Wait()

	return b
}

func alert(tittle string, text string) {
	// Create a new pushover app with a token
	app := pushover.New(appID)
	// Create a new recipient
	recipient := pushover.NewRecipient(groupID)
	// Create the message to send
	message := pushover.NewMessageWithTitle(text, tittle)
	// Send the message to the recipient
	if _, err := app.SendMessage(message, recipient); err != nil {
		log.Println(err.Error())
	}
}
