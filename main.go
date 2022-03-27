package main

import (
	"bytes"
	"encoding/binary"
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
	// Flags       []byte
	Flags   int16
	SortDst string
	Src     gopacket.Endpoint
	Dst     gopacket.Endpoint
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

func main() {
	sortings := make([]SortEvent, 0, 128)
	sortRequests := make([]SortRequest, 0, 128)

	handle, err := pcap.OpenOffline("20220324_RoboCap04.cap")
	check(err)
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		t, err := time.Parse("2006-01-02 15:04:05", "2022-03-24 16:45:00")
		check(err)

		if packet.Metadata().Timestamp.After(t) {
			handlePacket(packet, &sortings, &sortRequests)
		}
	}
	handle.Close()

	// for _, se := range sortings {
	// 	if se.CowName == 3162 {
	// 		ShowSortEvent(se)
	// 	}
	// }

	for i := 0; i < 100; i++ {
		// ShowSortRequest(sortRequests[i])
		ShowSortEvent(sortings[i])
		ExportSortEvents(&sortings, "test1.csv")
	}
}

func ShowSortRequest(sr SortRequest) {
	fmt.Printf("\n\n")
	fmt.Println("Time: ", sr.Time)
	fmt.Println(sr.Src, "->", sr.Dst)
	fmt.Println("Transponder: ", sr.Transponder)
	// printHex(sr.RawPayload)
}

func ShowSortEvent(se SortEvent) {
	fmt.Printf("\n\n")
	fmt.Println("Time: ", se.Time)
	fmt.Println(se.Src, "->", se.Dst)
	fmt.Println("Transponder: ", se.Transponder)
	fmt.Println("CowName: ", se.CowName)
	// fmt.Println("Flags: ", se.Flags)
	fmt.Println("Sorting to: ", se.SortDst)
	// printHex(se.RawPayload)
}

func ExportSortEvents(sortings *[]SortEvent, filename string) {
	// export format: time, transponder, cowname, destination
	f, err := os.Create(filename)
	check(err)
	defer f.Close()

	for _, se := range *sortings {
		fmt.Fprintf(f, "%v,%v,%v,%v\n", se.Time, se.Transponder, se.CowName, se.SortDst)
	}
}

func handlePacket(packet gopacket.Packet, sortings *[]SortEvent, sortRequests *[]SortRequest) {
	if udp := packet.Layer(layers.LayerTypeUDP); udp != nil && len(udp.LayerPayload()) > 4 {
		if udp.LayerPayload()[0] == 0x00 && udp.LayerPayload()[1] == 0x05 && udp.LayerPayload()[2] == 0x01 && udp.LayerPayload()[3] == 0x0a {
			if len(udp.LayerPayload()) == 18 && packet.Metadata().CaptureLength == 60 {
				sortRequest := decodeSortRequest(packet)
				*sortRequests = append(*sortRequests, sortRequest)
				return
			}
			if len(udp.LayerPayload()) == 222 && packet.Metadata().CaptureLength == 264 {
				sorting := decodeSortEvent(packet)
				*sortings = append(*sortings, sorting)
				return
			}
		}
	}
}

func GetSortingResult(se SortEvent) string {
	switch se.Dst.String() {
	case "172.17.172.201":
		if se.RawPayload[202] == 0x64 {
			return "Melkroboter 1"
		}
		switch se.Flags {
		case 0:
			return "Liegebox NL"
		case 128:
			return "Melkroboterbereich"
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
			return "Melkroboterbereich"
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
			return "Melkroboterbereich"
		default:
			panic("Unknown flag")
		}

	default:
		panic("Unknown destination")
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
	var sorting SortEvent
	payload := packet.Layer(layers.LayerTypeUDP).LayerPayload()

	sorting.RawPayload = payload
	sorting.Time = packet.Metadata().Timestamp
	sorting.Dst = packet.NetworkLayer().NetworkFlow().Dst()
	sorting.Src = packet.NetworkLayer().NetworkFlow().Src()
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
	// sorting.Flags = flagsRaw

	sorting.SortDst = GetSortingResult(sorting)

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
	fmt.Println(readable_timestamp, src, "->", dst, "Unix Miliseconds:", timestamp.UnixMilli())
	fmt.Printf("PAYLOAD: % x\n", payload)
	printHex(payload)
	fmt.Printf("\n")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
