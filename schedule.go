package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/amnonbc/aschedule/htmltable"
	"github.com/olekukonko/tablewriter"
)

const (
	tmFmt = "02/01/2006 15:04"
)

type participant struct {
	Timestamp   time.Time
	Email       string
	Name        string
	AgeRange    string
	Skype       string
	Disability  string
	Preferences []string

	// these fields are calculated by us when we schedule
	Assigned       string
	AssignedSlot   int
	AssignedTime   time.Time
	AssignedChoice int
}

type tabler interface {
	SetHeader([]string)
	Append(row []string)
	Render()
}

var startTime = time.Date(2021, 3, 6, 10, 0, 0, 0, time.UTC)
var startTimeKesia = time.Date(2021, 3, 5, 10, 0, 0, 0, time.UTC)

func numSlots(agent string) int {
	if agent == "Jo Williamson" {
		return 7
	}
	return 5
}

func loadSched(fn string) (particpants []participant) {
	f, err := os.Open(fn)
	if err != nil {
		log.Fatal(f)
	}
	defer f.Close()

	r := csv.NewReader(f)

	// skip headers
	r.Read()
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		tm, err := time.Parse("01/02/2006 15:14:05", row[0])

		p := participant{
			Timestamp:  tm,
			Email:      row[1],
			Name:       row[2],
			AgeRange:   row[3],
			Skype:      row[4],
			Disability: row[5],
		}
		for i := 6; i < len(row); i++ {
			p.Preferences = append(p.Preferences, row[i])
		}
		particpants = append(particpants, p)
	}
	return particpants
}

func getAgents(participants []participant) map[string][]participant {
	agents := make(map[string][]participant)
	for _, p := range participants {
		for _, a := range p.Preferences {
			agents[a] = nil
		}
	}
	return agents
}

func slotToTime(slot int, agent string) time.Time {
	start := startTime
	if strings.Contains(agent, "Friday") {
		start = startTimeKesia
	}
	distance := 30 * time.Minute
	if agent == "Jo Williamson" {
		distance = 20 * time.Minute

	}
	return start.Add(time.Duration(slot) * distance)
}

func (p participant) FormatTime() string {
	endTime := p.AssignedTime.Add(15 * (time.Minute))
	return p.AssignedTime.Format("15:04") + " - " + endTime.Format("15:04")
}

func (p participant) FormatDate() string {
	return p.AssignedTime.Format("Mon 2 Jan")
}

func schedule(agents map[string][]participant, participants []participant) {
	done := make(map[string]int)
	for i, p := range participants {
		if done[p.Email] > 1 {
			log.Println("Inoring entry", done[p.Email], "from", p.Name, p.Email)
			continue
		}

		for j, a := range p.Preferences {
			done[p.Email] += 1
			if len(agents[a]) < numSlots(a) {
				p.Assigned = a
				p.AssignedChoice = j + 1
				p.AssignedSlot = len(agents[a])
				p.AssignedTime = slotToTime(p.AssignedSlot, a)
				endTime := p.AssignedTime.Add(15 * (time.Minute))
				participants[i] = p
				agents[a] = append(agents[a], participants[i])
				log.Println("Scheduled", p.Timestamp.Format("15:04:05"), p.Name, p.Skype, a, p.AssignedTime.Weekday(),
					p.AssignedTime.Format(tmFmt), "-", endTime.Format("15:04"))
				break
			}
		}
		if p.Assigned == "" {
			log.Println("Could not find a match for", p.Name)
		}
	}
}

const emailTemplate = `
Dear {{ .Agent }},
Please find below a table with the schedule for your 1-1 meetings on {{ .Date }}. Just to remind you that each 
meeting lasts for 15 minutes, giving you a 15 minute break between each one.
With regards,
Festival Organisers
`

var agentEmailT = template.Must(template.New("email").Parse(emailTemplate))

func writeAgentBookings(tb tabler, bookings []participant) {
	tb.SetHeader([]string{"Time", "Name", "Age range", "Skype ID"})
	for _, p := range bookings {
		tb.Append([]string{
			p.AssignedTime.Format("15:04"),
			p.Name,
			p.AgeRange,
			strconv.Quote(p.Skype),
		})
	}
	tb.Render()
}

func dumpAgent(w io.Writer, tb tabler, a string, bookings []participant) {
	if len(bookings) == 0 {
		return
	}
	data := struct {
		Agent string
		Date  string
	}{
		Agent: a,
		Date:  bookings[0].AssignedTime.Format("Mon 2 Jan"),
	}
	agentEmailT.Execute(w, data)

	fmt.Fprintln(w)
	writeAgentBookings(tb, bookings)
}

func dumpAgents(w io.Writer, agents map[string][]participant) {
	for a, bookings := range agents {
		tb := tablewriter.NewWriter(os.Stdout)
		dumpAgent(w, tb, a, bookings)
	}
}

func dumpAllParticpants(table tabler, participants []participant) {
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Name < participants[j].Name
	})
	table.SetHeader([]string{"Name", "Agent", "Age Range", "slot", "n"})
	for _, p := range participants {
		if p.Assigned == "" {
			table.Append([]string{
				p.Name,
				"",
				p.AgeRange,
				"",
				"",
			})
			continue
		}
		table.Append([]string{
			p.Name,
			p.Assigned,
			p.AgeRange,
			p.AssignedTime.Format("Mon 15:04"),
			Ordinal(p.AssignedChoice),
		})
	}
	table.Render()
}

var toWriter = template.Must(template.ParseFiles("towriter.txt"))

func printLetterToWriter(w io.Writer, p participant) {
	toWriter.Execute(w, p)
}

func printAgentNumbers(table tabler, agents map[string][]participant, participants []participant) {
	choices := make(map[string][]int)
	for _, p := range participants {
		for i, c := range p.Preferences {
			if choices[c] == nil {
				choices[c] = make([]int, 5)
			}
			choices[c][i]++
		}
	}

	selected := make(map[string]int)
	for _, p := range participants {
		for _, c := range p.Preferences {
			selected[c]++
		}
	}

	table.SetHeader([]string{"Name", "selected", "1st", "2nd", "3rd", "4th", "5th"})
	for a, w := range agents {
		row := []string{
			a,
			strconv.Itoa(len(w)),
		}
		for _, c := range choices[a] {
			row = append(row, strconv.Itoa(c))
		}
		table.Append(row)
	}
	table.Render()

}

func printPeopleWithoutAgents(participants []participant) {
	without := 0
	fmt.Println("\nPeople without Agents")
	for _, p := range participants {
		if p.Assigned != "" {
			continue
		}
		fmt.Println(p.Name, p.Email)
		without++
	}

	fmt.Println(without, "writers without agents, out of", len(participants))
}

func Ordinal(num int) string {

	var ordinalDictionary = map[int]string{
		0: "th",
		1: "st",
		2: "nd",
		3: "rd",
		4: "th",
		5: "th",
		6: "th",
		7: "th",
		8: "th",
		9: "th",
	}

	// math.Abs() is to convert negative number to positive
	floatNum := math.Abs(float64(num))
	positiveNum := int(floatNum)

	if ((positiveNum % 100) >= 11) && ((positiveNum % 100) <= 13) {
		return "th"
	}

	return strconv.Itoa(num) + ordinalDictionary[positiveNum]

}

func printChoicesGot(participants []participant) {
	var got [6]int
	for _, p := range participants {
		got[p.AssignedChoice]++
	}

	for i := 1; i <= 5; i++ {
		fmt.Printf("%d people got their %s choice\n", got[i], Ordinal(i))
	}
	fmt.Printf("And %d people did not get any of their choices", got[0])
}

func loadPaid(fn string) (out []string) {

	f, err := os.Open(fn)
	if err != nil {
		log.Fatal(f)
	}
	defer f.Close()

	r := csv.NewReader(f)

	// skip headers
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if !strings.Contains(row[1], "@") {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		out = append(out, row[1])
	}
	return out
}

func reportMissing(paid []string, writers []participant) {
	wr := make(map[string]bool)
	for _, w := range writers {
		wr[strings.ToLower(w.Email)] = true
	}
	for _, p := range paid {
		if !wr[strings.ToLower(p)] {
			fmt.Println("Missing", p)
		}
	}
}

func dumpHTML(w http.ResponseWriter, r *http.Request) {
	particpants := getGSheetsData("1sizDYdctXcLyzO5g9TKiGmEWKBWdfM3nhUIrpX1dYWg")
	agents := getAgents(particpants)
	schedule(agents, particpants)
	io.WriteString(w, `
		<!DOCTYPE html>
		<html>
			<style>
			table, th, td {
				border: 1px solid black;
				border-collapse: collapse;
				padding: 5px;
				text-align: left;
			}
			</style>

		<title>Writers</title>
		<body>
		
		<h1>Writers</h1>
		<p></p>`)
	dumpAllParticpants(htmltable.NewWriter(w), particpants)

	io.WriteString(w, `<p>`)
	printAgentNumbers(htmltable.NewWriter(w), agents, particpants)

	for a, bookings := range agents {
		io.WriteString(w, `<p>`)
		dumpAgent(w, htmltable.NewWriter(w), a, bookings)
	}
	io.WriteString(w, `	
		</body>
		</html>
		`)
}

func main() {
	bf := flag.String("csv", "testdata/booking.csv", "booking file in csv format")
	spreadsheetId := flag.String("s", "1sizDYdctXcLyzO5g9TKiGmEWKBWdfM3nhUIrpX1dYWg",
		"Google Spreadsheet ID")
	payments := flag.String("pay", "", "csv of writers who have paid")
	web := flag.Bool("web", false, "open a web server")
	flag.Parse()

	if *web {
		http.HandleFunc("/", dumpHTML)
		log.Println("listening on :8888")
		log.Fatal(http.ListenAndServe(":8888", nil))
	}

	particpants := []participant{}
	if *bf != "" {
		particpants = loadSched(*bf)
	}

	particpants = getGSheetsData(*spreadsheetId)
	agents := getAgents(particpants)

	if *payments != "" {
		paid := loadPaid(*payments)
		reportMissing(paid, particpants)
		return
	}

	schedule(agents, particpants)

	//for _, p := range particpants {
	//	printLetterToWriter(os.Stdout, p)
	//}

	dumpAgents(os.Stdout, agents)

	dumpAllParticpants(tablewriter.NewWriter(os.Stdout), particpants)

	printAgentNumbers(tablewriter.NewWriter(os.Stdout), agents, particpants)

	printPeopleWithoutAgents(particpants)

	printChoicesGot(particpants)
}

// TODO: configure email using https://support.google.com/mail/answer/7126229?visit_id=637480509797127588-3781772115&hl=en&rd=1
