package cloudalert

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gregdel/pushover"
)

/*Quotes - ..
 */
type Quotes []Quote

/*Quote - ..
 */
type Quote struct {
	Symbol           string  `json:"_symbol"`
	AskPrice         float64 `json:"_ask_price"`
	BidPrice         float64 `json:"_bid_price"`
	RefBidPrice      float64 `json:"_ref_bid_price"`
	HighBidPrice     float64 `json:"_high_bid_price"`
	LowBidPrice      float64 `json:"_low_bid_price"`
	MidPrice         float64 `json:"_mid_price"`
	BidDayChange     float64 `json:"_bid_day_change"`
	BidDayChangePcnt string  `json:"_bid_day_change_pcnt"`
	QuoteTm          int64   `json:"_quote_tm"`
	Pips             float64 `json:"_pips"`
	PipsLot          float64 `json:"_pips_lot"`
	Digits           float64 `json:"_digits"`
	MonthMin         float64 `json:"_30d_min_bid_price"`
	MonthMax         float64 `json:"_30d_max_bid_price"`
}

var (
	asset   = os.Getenv("ASSET")
	h       = os.Getenv("HIGH")
	l       = os.Getenv("LOW")
	t       = os.Getenv("TARGET")
	appID   = os.Getenv("APP_ID")
	groupID = os.Getenv("GROUP_ID")
	city    = "Europe/Warsaw"
	apiURL  = fmt.Sprintf("https://api.30.bossa.pl/API/FX/v1/SYMBOLS/%s.", asset)
	// http.Clients should be reused instead of created as needed.
	client = &http.Client{
		Timeout: 3 * time.Second,
	}
)

func init() {
}

func main() {
}

/*CloudAlert - ..
 */
func CloudAlert(w http.ResponseWriter, r *http.Request) {
	var high, low, target float64
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
	if tq := query.Get("t"); tq != "" {
		target, _ = strconv.ParseFloat(tq, 64)
	} else {
		target, _ = strconv.ParseFloat(t, 64)
	}
	if a := query.Get("a"); a != "" {
		asset = a
	}
	bid := processSignals(&high, &low, &target)
	w.WriteHeader(http.StatusOK)
	response := bid
	if err := json.NewEncoder(w).Encode(&response); err != nil {
		log.Println(err.Error())
	}
}

/*processSignals - gets data from BOŚ API and logic of signals
This is stateless function
*/
func processSignals(high *float64, low *float64, target *float64) string {
	var b string
	if response, err := client.Get(apiURL); err != nil {
		log.Fatalln(err.Error())
	} else {
		var body Quotes
		location, _ := time.LoadLocation(city)
		defer response.Body.Close()
		json.NewDecoder(response.Body).Decode(&body)
		tm := time.Unix(0, body[0].QuoteTm*int64(time.Millisecond))
		bid := body[0].BidPrice
		chng := body[0].BidDayChange
		pct := body[0].BidDayChangePcnt
		b = fmt.Sprintf("%s - Bid: %.2f Change: %.2f %s", tm.In(location).Format("15:04:05"), bid, chng, pct)
		// Main logic loop
		if math.Abs(*target-bid) < 2.00 {
			msg := fmt.Sprintf("%s is now at %.2f", asset, bid)
			sendAlert(msg, "Closing in on target price")
		} else if bid > *high {
			msg := fmt.Sprintf("%s is now at %.2f", asset, bid)
			sendAlert(msg, "Making money")
		} else if bid < *low {
			msg := fmt.Sprintf("%s is now at %.2f", asset, bid)
			sendAlert(msg, "Losing money!")
		} else if math.Abs(chng) > 2.00 {
			msg := fmt.Sprintf("%s is now at %.2f, %s", asset, bid, pct)
			sendAlert(msg, "Big move today!")
		} else if max30 := body[0].MonthMax; (max30 - bid) < 2.00 {
			msg := fmt.Sprintf("%s is now at %.2f, %s", asset, bid, pct)
			sendAlert(msg, "Melting up!")
		} else if min30 := body[0].MonthMin; (bid - min30) < 0.50 {
			msg := fmt.Sprintf("%s is now at %.2f, %s", asset, bid, pct)
			sendAlert(msg, "Melting down!")
		}
	}
	return b
}

func sendAlert(tittle string, text string) {
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
