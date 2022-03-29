package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type SortEvent struct {
	Time        time.Time
	Index       int16
	CowName     int16
	Transponder int32
	RawPayload  []byte
	Flags       int16
	SortSrc     BarnLocation
	SortDst     BarnLocation
	Gate        Gate
	IpSrc       gopacket.Endpoint
	IpDst       gopacket.Endpoint
}

type SortRequest struct {
	Time        time.Time
	Index       int16
	Transponder int32
	RawPayload  []byte
	Src         gopacket.Endpoint
	Dst         gopacket.Endpoint
	Num1        byte
}

type BarnLocation struct {
	Id   int
	Name string
}

type Gate struct {
	Id   int
	Name string
}

func main() {
	createNewDb := flag.Bool("createdb", false, "Use this flag if a new database should be created")
	dbName := flag.String("db", "testdb01.db", "Path to the database")
	flag.Parse()

	if *createNewDb {
		createDb(*dbName)
	}

	db := openDb(*dbName)
	defer db.Close()

	srChan := make(chan SortEvent, 1e2)

	go SaveAndShowSE(srChan, db)

	// pcapIn, err := pcap.OpenLive("eth0", 400, true, pcap.BlockForever)
	pcapIn, err := pcap.OpenOffline("20220320_RoboCap03.cap")
	check(err)
	packetSource := gopacket.NewPacketSource(pcapIn, pcapIn.LinkType())

	go handlePacket(packetSource.Packets(), srChan)

	for {
		time.Sleep(100 * time.Second)
	}
}

func SaveAndShowSE(seIn <-chan SortEvent, db *sql.DB) {
	for {
		se := <-seIn
		insertSortEvent(se, db)
		ShowSortEvent(se)
	}
}

func ShowSortRequest(sr SortRequest) {
	fmt.Printf("\n\n")
	fmt.Println("Time: ", sr.Time.Format("2006-01-02 15:04:05"))
	fmt.Println(sr.Src, "->", sr.Dst)
	fmt.Println("Transponder: ", sr.Transponder)
}

func ShowSortEvent(se SortEvent) {
	fmt.Printf("\n\n")
	fmt.Println("Time: ", se.Time.Format("2006-01-02 15:04:05"))
	fmt.Println(se.IpSrc, "->", se.IpDst)
	fmt.Println("Transponder: ", se.Transponder)
	fmt.Println("CowName: ", se.CowName)
	fmt.Println("At gate: ", se.Gate)
	fmt.Println("Coming from: ", se.SortSrc)
	fmt.Println("Sorting to: ", se.SortDst)
}

func ExportSortEvent(se SortEvent, f *os.File) {
	fmt.Fprintf(f, "%v,%v,%v,%v,%v,%v\n", se.Time, se.Transponder, se.CowName, se.SortSrc.Id, se.SortDst.Id, se.Gate.Id)
}

func handlePacket(packetsChan <-chan gopacket.Packet, srChan chan<- SortEvent) {
	for {
		packet := <-packetsChan

		if udp := packet.Layer(layers.LayerTypeUDP); udp != nil && len(udp.LayerPayload()) > 4 {
			if udp.LayerPayload()[0] == 0x00 && udp.LayerPayload()[1] == 0x05 && udp.LayerPayload()[2] == 0x01 && udp.LayerPayload()[3] == 0x0a {
				if len(udp.LayerPayload()) == 18 && packet.Metadata().CaptureLength == 60 {
					continue
				}
				if len(udp.LayerPayload()) == 222 && packet.Metadata().CaptureLength == 264 {
					se := decodeSortEvent(packet)
					if se.RawPayload[202] != 0x64 {
						srChan <- se
						continue
					}
				}
			}
		}
	}
}

func GetSortingResult(se SortEvent) string {
	switch se.IpDst.String() {
	case "172.17.172.201":
		if se.RawPayload[202] == 0x64 {
			return "Melkroboter 1"
		}
		switch se.Flags {
		case 0:
			return "Liegebox NL"
		case 128:
			return "Melkbereich"
		default:
			panic("Unknown flag")
		}

	case "172.17.172.202":
		if se.RawPayload[202] == 0x64 {
			return "Melkroboter 2"
		}
		switch se.Flags {
		case 0:
			return "Liegebox HL"
		case 256:
			return "Melkbereich"
		case 512:
			return "Liegebox NL"
		default:
			panic("Unknown flag")
		}

	case "172.17.172.203":
		if se.RawPayload[202] == 0x64 {
			return "Melkroboter 3"
		}
		panic("Roboter with ip ..203 can only sort to Melkroboter 3")

	case "172.17.172.204":
		if se.RawPayload[202] == 0x64 {
			return "Melkroboter 4"
		}
		switch se.Flags {
		case 0:
			return "Liegebox HL"
		case 128:
			return "Melkbereich"
		default:
			panic("Unknown flag")
		}

	default:
		panic("Unknown destination")
	}
}

func IpToGate(ip string) string {
	switch ip {
	case "172.17.172.201":
		return "Gate NL"

	case "172.17.172.202":
		return "Gate Ausgang Melkbereich"

	case "172.17.172.204":
		return "Gate HL"

	default:
		panic("Invalid ip")
	}
}

func GateToOrigin(gate string) string {
	switch gate {
	case "Gate Ausgang Melkbereich":
		return "Melkbereich"

	case "Gate NL":
		return "Fressbereich NL"

	case "Gate HL":
		return "Fressbereich HL"

	default:
		panic("Unknown gate")
	}
}

func decodeSortRequest(packet gopacket.Packet) SortRequest {
	var sortRequest SortRequest
	payload := packet.Layer(layers.LayerTypeUDP).LayerPayload()

	sortRequest.RawPayload = payload
	sortRequest.Time = packet.Metadata().Timestamp
	sortRequest.Dst = packet.NetworkLayer().NetworkFlow().Dst()
	sortRequest.Src = packet.NetworkLayer().NetworkFlow().Src()
	sortRequest.Num1 = payload[17]

	indexRaw := payload[4:6]
	buf := bytes.NewReader(indexRaw)
	err := binary.Read(buf, binary.BigEndian, &sortRequest.Index)
	check(err)

	transponderRaw := payload[12:16]
	buf = bytes.NewReader(transponderRaw)
	err = binary.Read(buf, binary.BigEndian, &sortRequest.Transponder)
	check(err)

	return sortRequest
}

func decodeSortEvent(packet gopacket.Packet) SortEvent {
	BarnLocationNameToId := map[string]int{
		// "ERROR: Invalid":     0,
		"Melkroboter 1":   1,
		"Liegebox NL":     2,
		"Melkbereich":     3,
		"Melkroboter 2":   4,
		"Liegebox HL":     5,
		"Melkroboter 3":   6,
		"Melkroboter 4":   7,
		"Fressbereich NL": 8,
		"Fressbereich HL": 9,
	}

	GateNameToId := map[string]int{
		// "ERROR: Invalid":           0,
		"Gate NL":                  1,
		"Gate HL":                  2,
		"Gate Ausgang Melkbereich": 3,
	}

	var sorting SortEvent
	payload := packet.Layer(layers.LayerTypeUDP).LayerPayload()

	sorting.RawPayload = payload
	sorting.Time = packet.Metadata().Timestamp
	sorting.IpDst = packet.NetworkLayer().NetworkFlow().Dst()
	sorting.IpSrc = packet.NetworkLayer().NetworkFlow().Src()
	cowNameRaw := payload[20:22]
	transponderRaw := payload[12:16]
	indexRaw := payload[4:6]
	flagsRaw := payload[194:196]

	buf := bytes.NewReader(cowNameRaw)
	err := binary.Read(buf, binary.BigEndian, &sorting.CowName)
	check(err)

	buf = bytes.NewReader(transponderRaw)
	err = binary.Read(buf, binary.BigEndian, &sorting.Transponder)
	check(err)

	buf = bytes.NewReader(indexRaw)
	err = binary.Read(buf, binary.BigEndian, &sorting.Index)
	check(err)

	buf = bytes.NewReader(flagsRaw)
	err = binary.Read(buf, binary.BigEndian, &sorting.Flags)
	check(err)

	sorting.SortDst.Name = GetSortingResult(sorting)
	sorting.SortDst.Id = BarnLocationNameToId[sorting.SortDst.Name]

	if sorting.IpDst.String() != "172.17.172.203" {
		sorting.Gate.Name = IpToGate(sorting.IpDst.String())
		sorting.Gate.Id = GateNameToId[sorting.Gate.Name]

		sorting.SortSrc.Name = GateToOrigin(sorting.Gate.Name)
		sorting.SortSrc.Id = BarnLocationNameToId[sorting.SortSrc.Name]
	}

	return sorting
}

func printHex(data []byte) {
	for _, b := range data {
		// fmt.Printf("%v:", i)
		if b == 0x0 {
			fmt.Printf("\033[38;5;%dm", 240)
			fmt.Printf("%02x ", b)
			fmt.Printf("\033[0m")
		} else {
			fmt.Printf("%02x ", b)
		}
		// fmt.Printf("\n")
	}
}

func ShowPacketInfo(packet gopacket.Packet) {
	timestamp := packet.Metadata().Timestamp
	readable_timestamp := timestamp.Format("2006-01-02 15:04:05")
	dst := packet.NetworkLayer().NetworkFlow().Dst()
	src := packet.NetworkLayer().NetworkFlow().Src()
	payload := packet.Layer(layers.LayerTypeUDP).LayerPayload()

	fmt.Printf("\n\n")
	fmt.Println(readable_timestamp, src, "->", dst)
	fmt.Printf("PAYLOAD: % x\n", payload)
	printHex(payload)
	fmt.Printf("\n")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
