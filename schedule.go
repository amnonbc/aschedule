package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
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
	return start.Add(time.Duration(slot-1) * 30 * time.Minute)
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

		for _, a := range p.Preferences {
			done[p.Email] += 1
			if len(agents[a]) < numSlots {
				p.Assigned = a
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

func dumpAgents(agents map[string][]participant) {
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
				p.AgeRange,
				strconv.Quote(p.Skype),
			})
		}
		fmt.Println(a)
		t.Execute(os.Stdout, data)
		table.Render()
		fmt.Println()
	}
}

func dumpAllParticpants(participants []participant) {
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Name < participants[j].Name
	})
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "email", "Agent", "Age Range", "slot"})
	for _, p := range participants {
		table.Append([]string{
			p.Name,
			p.Email,
			p.Assigned,
			p.AgeRange,
			p.AssignedTime.Format("Mon 15:04"),
		})
	}
	table.Render()
}

var toWriter = template.Must(template.ParseFiles("towriter.txt"))

func printLetterToWriter(p participant) {
	toWriter.Execute(os.Stdout, p)
}

func printAgentNumbers(agents map[string][]participant, participants []participant) {
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

	table := tablewriter.NewWriter(os.Stdout)
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

	for _, p := range particpants {
		printLetterToWriter(p)
	}

	dumpAgents(agents)

	dumpAllParticpants(particpants)

	printAgentNumbers(agents, particpants)

	printPeopleWithoutAgents(particpants)
}
