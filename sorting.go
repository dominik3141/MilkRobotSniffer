package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
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
	DstIsRobo   bool
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

func GetSortingResult(se SortEvent) string {
	switch se.IpDst.String() {
	case "172.17.172.201":
		if se.DstIsRobo {
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
		if se.DstIsRobo {
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
		if se.DstIsRobo {
			return "Melkroboter 3"
		}
		panic("Roboter with ip ..203 can only sort to Melkroboter 3")

	case "172.17.172.204":
		if se.DstIsRobo {
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
		"Melkroboter 1":            4,
		"Melkroboter 2":            5,
		"Melkroboter 3":            6,
		"Melkroboter 4":            7,
	}

	var se SortEvent
	payload := packet.Layer(layers.LayerTypeUDP).LayerPayload()

	if payload[202] != 0x64 {
		se.DstIsRobo = false
	} else { // for sortings at the roboter entrance
		se.DstIsRobo = true
	}

	se.RawPayload = payload
	se.Time = packet.Metadata().Timestamp
	se.IpDst = packet.NetworkLayer().NetworkFlow().Dst()
	se.IpSrc = packet.NetworkLayer().NetworkFlow().Src()
	cowNameRaw := payload[20:22]
	transponderRaw := payload[12:16]
	indexRaw := payload[4:6]
	flagsRaw := payload[194:196]

	buf := bytes.NewReader(cowNameRaw)
	err := binary.Read(buf, binary.BigEndian, &se.CowName)
	check(err)

	buf = bytes.NewReader(transponderRaw)
	err = binary.Read(buf, binary.BigEndian, &se.Transponder)
	check(err)

	buf = bytes.NewReader(indexRaw)
	err = binary.Read(buf, binary.BigEndian, &se.Index)
	check(err)

	buf = bytes.NewReader(flagsRaw)
	err = binary.Read(buf, binary.BigEndian, &se.Flags)
	check(err)

	se.SortDst.Name = GetSortingResult(se)
	se.SortDst.Id = BarnLocationNameToId[se.SortDst.Name]

	if !se.DstIsRobo {
		se.Gate.Name = IpToGate(se.IpDst.String())
		se.Gate.Id = GateNameToId[se.Gate.Name]

		se.SortSrc.Name = GateToOrigin(se.Gate.Name)
		se.SortSrc.Id = BarnLocationNameToId[se.SortSrc.Name]
	}

	return se
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
	fmt.Println("At gate: ", se.Gate.Name)
	fmt.Println("Coming from: ", se.SortSrc.Name)
	fmt.Println("Sorting to: ", se.SortDst.Name)
	fmt.Println("DstIsRobot: ", se.DstIsRobo)
}

func ExportSortEvent(se SortEvent, f *os.File) {
	fmt.Fprintf(f, "%v,%v,%v,%v,%v,%v\n", se.Time, se.Transponder, se.CowName, se.SortSrc.Id, se.SortDst.Id, se.Gate.Id)
}
