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
	asset         = os.Getenv("ASSET")
	h             = os.Getenv("HIGH")
	l             = os.Getenv("LOW")
	t             = os.Getenv("TARGET")
	appID         = os.Getenv("APP_ID")
	groupID       = os.Getenv("GROUP_ID")
	meltUp, _     = strconv.ParseFloat(os.Getenv("MELTUP"), 64)
	meltDown, _   = strconv.ParseFloat(os.Getenv("MELTDOWN"), 64)
	largeMove, _  = strconv.ParseFloat(os.Getenv("LARGE_MOVE"), 64)
	targetZone, _ = strconv.ParseFloat(os.Getenv("TARGET_ZONE"), 64)
	// http.Clients should be reused instead of created as needed.
	client = &http.Client{
		Timeout: 3 * time.Second,
	}
	userAgent   = randUserAgent()
	city        = os.Getenv("CITY")
	location, _ = time.LoadLocation(city)
	// Create a new pushover app with a token
	app = pushover.New(appID)
	// Create a new recipient
	recipient = pushover.NewRecipient(groupID)
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
	apiURL := fmt.Sprintf("%s%s.", os.Getenv("API_URL"), asset)
	request, _ := http.NewRequest("GET", apiURL, nil)
	request.Header.Set("User-Agent", userAgent)
	if response, err := client.Do(request); err != nil {
		log.Fatalln(err.Error())
	} else {
		var body Quotes
		json.NewDecoder(response.Body).Decode(&body)
		tm := time.Unix(0, body[0].QuoteTm*int64(time.Millisecond))
		bid := body[0].BidPrice
		chng := body[0].BidDayChange
		pct := body[0].BidDayChangePcnt
		h := body[0].HighBidPrice
		l := body[0].LowBidPrice
		b = fmt.Sprintf("%s - Bid: %.2f Change: %.2f %s", tm.In(location).Format("15:04:05"), bid, chng, pct)
		// Main logic loop
		if math.Abs(*target-bid) < targetZone {
			msg := fmt.Sprintf("%s is now at %.2f", asset, bid)
			sendAlert(msg, "Closing in on target price", pushover.PriorityEmergency, tm)
		} else if bid > *high {
			msg := fmt.Sprintf("%s is now at %.2f", asset, bid)
			sendAlert(msg, "Above higher band", pushover.PriorityNormal, tm)
		} else if bid < *low {
			msg := fmt.Sprintf("%s is now at %.2f", asset, bid)
			sendAlert(msg, "Below lower band", pushover.PriorityNormal, tm)
		} else if (h - l) > largeMove {
			msg := fmt.Sprintf("%s is now at %.2f, %s", asset, bid, pct)
			sendAlert(msg, "Big volatility today!", pushover.PriorityHigh, tm)
		} else if math.Abs(chng) > largeMove {
			msg := fmt.Sprintf("%s is now at %.2f, %s", asset, bid, pct)
			sendAlert(msg, "Big move today!", pushover.PriorityHigh, tm)
		}
		// else if max30 := body[0].MonthMax; (max30 - bid) < meltUp {
		// 	msg := fmt.Sprintf("%s is now at %.2f, %s", asset, bid, pct)
		// 	sendAlert(msg, "Melting up!", pushover.PriorityHigh, tm)
		// } else if min30 := body[0].MonthMin; (bid - min30) < meltDown {
		// 	msg := fmt.Sprintf("%s is now at %.2f, %s", asset, bid, pct)
		// 	sendAlert(msg, "Melting down!", pushover.PriorityHigh, tm)
		// }
	}
	return b
}

func sendAlert(msgText string, title string, priority int, ts time.Time) {
	webpageURL := fmt.Sprintf("%s?a=%s", os.Getenv("WEB_URL"), asset)
	// Create the message to send
	message := pushover.Message{
		Message:   msgText,
		Title:     title,
		Priority:  priority,
		URL:       webpageURL,
		URLTitle:  fmt.Sprintf("Chart %s", asset),
		Timestamp: ts.Unix(),
	}
	if priority == 2 {
		message.Sound = pushover.SoundIncoming
		message.Retry = 60 * time.Second
		message.Expire = 4 * time.Minute
	} else if priority == 1 {
		message.Sound = pushover.SoundCashRegister
	} else {
		message.Sound = pushover.SoundVibrate
	}
	// Send the message to the recipient
	if _, err := app.SendMessage(&message, recipient); err != nil {
		log.Println(err.Error())
	}
}
