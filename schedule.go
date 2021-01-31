package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

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
	Assigned     string
	AssignedSlot int
	AssignedTime time.Time
}

var numSlots = 5
var startTime = time.Date(2021, 3, 6, 10, 0, 0, 0, time.UTC)
var startTimeKesia = time.Date(2021, 3, 5, 10, 0, 0, 0, time.UTC)

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

func getAgents(participants []participant) map[string][]*participant {
	agents := make(map[string][]*participant)
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
	return start.Add(time.Duration(slot-1) * 30 * time.Minute)
}

func schedule(agents map[string][]*participant, participants []participant) {
	for i, p := range participants {
		for _, a := range p.Preferences {

			if len(agents[a]) < numSlots {
				p.Assigned = a
				agents[a] = append(agents[a], &participants[i])
				p.AssignedSlot = len(agents[a])
				p.AssignedTime = slotToTime(p.AssignedSlot, a)
				endTime := p.AssignedTime.Add(30 * (time.Minute))
				participants[i] = p
				log.Println("Scheduled", p.Name, p.Skype, a, p.AssignedTime.Weekday(),
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

func dumpAgents(agents map[string][]*participant, participants []participant) {
	t, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		log.Fatal(err)
	}

	for a, bookings := range agents {
		if len(bookings) == 0 {
			continue
		}
		data := struct {
			Agent string
			Date  string
		}{
			Agent: a,
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Time", "Name", "Age range", "Skype ID"})
		for _, p := range bookings {
			data.Date = p.AssignedTime.Format("Mon 2 Jan")
			table.Append([]string{
				p.AssignedTime.Format("15:04"),
				p.Name,
				"(" + p.AgeRange + ")",
				strconv.Quote(p.Skype),
			})
		}
		fmt.Println(a)
		t.Execute(os.Stdout, data)
		table.Render()
		fmt.Println()
	}
}

func main() {
	bf := flag.String("csv", "testdata/booking.csv", "booking file in csv format")
	spreadsheetId := flag.String("s", "1sizDYdctXcLyzO5g9TKiGmEWKBWdfM3nhUIrpX1dYWg",
		"Google Spreadsheet ID")
	flag.Parse()

	particpants := []participant{}
	if *bf != "" {
		particpants = loadSched(*bf)
	}

	particpants = getGSheetsData(*spreadsheetId)
	agents := getAgents(particpants)

	schedule(agents, particpants)

	dumpAgents(agents, particpants)
}
