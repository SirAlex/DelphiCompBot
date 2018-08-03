package main

import (
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RuBoardStats struct {
	RegistrationDate   time.Time
	PointsSinceRegDate int
	PointsWarez        int
	TotalPoints        int
}

func datediff(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour = int(h2 - h1)
	min = int(m2 - m1)
	sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}

	return
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func GetUserPoints(user string) RuBoardStats {
	var rz RuBoardStats

	// Declare http client
	var Url *url.URL
	Url, err := url.Parse("http://forum.ru-board.com")
	if err != nil {
		panic("boom")
	}

	Url.Path += "/profile.cgi"
	parameters := url.Values{}
	parameters.Add("action", "show")
	parameters.Add("member", user)
	Url.RawQuery = parameters.Encode()

	fmt.Printf("Encoded URL is %q\n", Url.String())

	client := &http.Client{}

	// Declare HTTP Method and Url
	req, err := http.NewRequest("GET", Url.String(), nil)

	// Set cookie
	req.Header.Set("Cookie",
		"amembernamecookie="+os.Getenv("RBLOGIN")+"; "+
			"apasswordcookie="+os.Getenv("RBPASS"))
	resp, err := client.Do(req)

	// Read response
	tr := transform.NewReader(resp.Body, charmap.Windows1251.NewDecoder())
	data, err := ioutil.ReadAll(tr)
	body := string(data)

	// error handle
	if err != nil {
		fmt.Printf("error = %s \n", err)
	}

	// Find registration date
	re := regexp.MustCompile("Дата регистрации:</b></td><td>(\\d{2}-\\d{2}-\\d{4})</td></tr>")
	matches := re.FindStringSubmatch(body)
	if (len(matches) == 2) && (matches[1] != "") {
		rz.RegistrationDate, _ = time.Parse("02-01-2006", matches[1])
		var year int
		year, rz.PointsSinceRegDate, _, _, _, _ = datediff(time.Now(), rz.RegistrationDate)
		rz.PointsSinceRegDate += year * 12
	}

	// Find warez points
	re = regexp.MustCompile("Варезник</b></a></td>\\n[ ]+<td><b>(\\d+)</b></td>")
	matches = re.FindStringSubmatch(body)
	if (len(matches) == 2) && (matches[1] != "") {
		rz.PointsWarez, _ = strconv.Atoi(matches[1])
	}

	rz.TotalPoints = rz.PointsSinceRegDate + rz.PointsWarez

	return rz
}

func SendPrivateMessage(user string, subject string, body string) {
	// Declare http client
	var Url *url.URL
	Url, err := url.Parse("http://forum.ru-board.com")
	if err != nil {
		panic("boom")
	}

	Url.Path += "/messanger.cgi"
	parameters := url.Values{}
	parameters.Add("aboock", "Адресная книга")
	parameters.Add("action", "send")
	parameters.Add("touser", user)
	parameters.Add("message", body)
	parameters.Add("msgtitle", subject)

	fmt.Printf("Encoded URL is %q, Params: %q\n", Url.String(), parameters.Encode())

	client := &http.Client{}

	// Declare HTTP Method and Url
	req, err := http.NewRequest("POST", Url.String(), strings.NewReader(parameters.Encode()))

	// Set cookie
	req.Header.Set("Cookie",
		"amembernamecookie="+os.Getenv("RBLOGIN")+"; "+
			"apasswordcookie="+os.Getenv("RBPASS"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(parameters.Encode())))

	resp, err := client.Do(req)

	// Read response
	tr := transform.NewReader(resp.Body, charmap.Windows1251.NewDecoder())
	data, err := ioutil.ReadAll(tr)
	respbody := string(data)

	fmt.Println(respbody)

	// error handle
	if err != nil {
		fmt.Printf("error = %s \n", err)
	}
}
