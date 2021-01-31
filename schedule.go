package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

const tmFmt = "02/01/2006 15:04"


type participant struct {
	Id int
	Timestamp time.Time
	Email string
	Name string
	AgeRange string
	Skype string
	Disability string
	Preferences []string
	Assigned string
	AssignedSlot int
	AssignedTime time.Time
}

var numSlots = 5
var startTime = time.Date(2020, 3, 6, 10, 0,0,0,time.UTC)
var startTimeKesia = time.Date(2020, 3, 5, 10, 0,0,0,time.UTC)

func loadSched(f io.Reader) (particpants []participant) {
	r := csv.NewReader(f)

	// skip headers
	r.Read()
	id := 0
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
			Id : id,
			Timestamp:   tm,
			Email:       row[1],
			Name:        row[2],
			AgeRange:    row[3],
			Skype: row[4],
			Disability:    row[5],
		}
		id++
		for i := 6; i < len(row); i++ {
			p.Preferences = append(p.Preferences, row[i])
		}
		particpants = append(particpants, p)
	}
	return particpants
}


func getAgents(participants []participant) map[string][]int {
	agents := make(map[string][]int)
	for _, p := range participants {
		for _, a := range p.Preferences {
			agents[a] = nil
		}
	}
	return agents
}

func slotToTime( slot int, agent string) time.Time{
	start := startTime
	if strings.Contains(agent, "Friday") {
		start = startTimeKesia
	}
	return start.Add(time.Duration(slot-1) * 30 * time.Minute)
}

func schedule(agents map[string][]int, participants []participant) {
	for i, p := range participants {
		for _, a := range p.Preferences {

			if len(agents[a]) < numSlots {
				p.Assigned = a
				agents[a] = append(agents[a], p.Id)
				p.AssignedSlot = len(agents[a])
				p.AssignedTime = slotToTime(p.AssignedSlot, a)
				endTime :=p.AssignedTime.Add(30*(time.Minute))
				participants[i] = p
				//endTime := p.AssignedTime.Add(30*time.Minute)
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

func dumpAgents(agents map[string][]int, participants []participant) {
	for a, bookings := range agents {
		fmt.Println("\n", a)
		for _, b := range bookings {
			fmt.Println("  ", participants[b].AssignedTime.Format("15:04"), participants[b].Name)
		}
	}
}

func main() {
	bf := flag.String("b", "testdata/booking.csv", "booking file in csv format")
	flag.Parse()

	f, err := os.Open(*bf)
	if err != nil {
		log.Fatal(f)
	}
	defer f.Close()
	particpants := loadSched(f)
	agents := getAgents(particpants)

	schedule(agents, particpants)

	dumpAgents(agents, particpants)
}
