package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	Flags       []byte
	Src         gopacket.Endpoint
	Dst         gopacket.Endpoint
}

type SortRequest struct {
	Time        time.Time
	Index       int16
	Transponder int32
	RawPayload  []byte
	Src         gopacket.Endpoint
	Dst         gopacket.Endpoint
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
			handlePacket(packet, sortings, sortRequests)
		}
	}
	handle.Close()

	for _, sorting := range sortings {
		if sorting.CowName == 3055 {
			fmt.Printf("\n\n")
			fmt.Println("Time: ", sorting.Time)
			fmt.Println("Transponder: ", sorting.Transponder)
			fmt.Println("CowName: ", sorting.CowName)
			printHex(sorting.RawPayload)
		}
	}
}

func handlePacket(packet gopacket.Packet, sortings []SortEvent, sortRequests []SortRequest) {
	if udp := packet.Layer(layers.LayerTypeUDP); udp != nil {
		if udp.LayerPayload()[0] == 0x00 && udp.LayerPayload()[1] == 0x05 && udp.LayerPayload()[2] == 0x01 && udp.LayerPayload()[3] == 0x0a {
			if len(udp.LayerPayload()) == 18 {
				sortRequest := decodeSortRequest(packet)
				sortRequests = append(sortRequests, sortRequest)
				return
			}
			if len(udp.LayerPayload()) > 21 {
				sorting := decodeSortEvent(packet)
				sortings = append(sortings, sorting)
				return
			}
		}
	}
}

func decodeSortRequest(packet gopacket.Packet) SortRequest {
	var sortRequest SortRequest
	payload := packet.Layer(layers.LayerTypeUDP).LayerPayload()

	sortRequest.RawPayload = payload
	sortRequest.Time = packet.Metadata().Timestamp
	sortRequest.Dst = packet.NetworkLayer().NetworkFlow().Dst()
	sortRequest.Src = packet.LinkLayer().LinkFlow().Src()

	indexRaw := payload[4:5]
	buf := bytes.NewReader(indexRaw)
	err := binary.Read(buf, binary.BigEndian, &sortRequest.Index)
	check(err)

	return sortRequest
}

func decodeSortEvent(packet gopacket.Packet) SortEvent {
	var sorting SortEvent
	payload := packet.Layer(layers.LayerTypeUDP).LayerPayload()

	sorting.RawPayload = payload
	sorting.Time = packet.Metadata().Timestamp
	sorting.Dst = packet.NetworkLayer().NetworkFlow().Dst()
	sorting.Src = packet.LinkLayer().LinkFlow().Src()
	cowNameRaw := payload[20:22]
	transponderRaw := payload[12:16]
	indexRaw := payload[4:5]

	buf := bytes.NewReader(cowNameRaw)
	err := binary.Read(buf, binary.BigEndian, &sorting.CowName)
	check(err)

	buf = bytes.NewReader(transponderRaw)
	err = binary.Read(buf, binary.BigEndian, &sorting.Transponder)
	check(err)

	buf = bytes.NewReader(indexRaw)
	err = binary.Read(buf, binary.BigEndian, &sorting.Index)
	check(err)

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
