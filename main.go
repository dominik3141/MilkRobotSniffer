package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
)

var verboseFlag *bool
var takePictures *bool
var savePcap *bool

func main() {
	// command line arguments
	createNewDb := flag.Bool("createdb", false, "Use this flag if a new database should be created")
	verboseFlag = flag.Bool("v", true, "Print sort events and stays to the command line")
	takePictures = flag.Bool("p", true, "Take pictures of the sortings using the ip cameras")
	savePcap = flag.Bool("w", true, "Wheter to save a pcap file containing the unfiltered raw packets")
	dbName := flag.String("db", "testdb02.db", "Path to the database")
	flag.Parse()

	// database stuff
	if *createNewDb {
		createDb(*dbName)
	}
	db := openDb(*dbName)
	defer db.Close()

	// create channels
	srChan := make(chan SortEvent, 1e2)
	seToStaysChan := make(chan SortEvent, 1e2)
	staysToSaveChan := make(chan Stay, 1e2)
	seForPicture := make(chan SortEvent, 1e2)

	// start goroutine for saving and displaying sorting events
	go SaveAndShowSE(srChan, db, seToStaysChan, seForPicture)

	// start goroutine for saving pictures of the sorting events
	// if *takePictures {
	// 	go takePictureRoutine(seForPicture)
	// }

	// analyze the SortEvents and convert them into stays
	go SortingResultsToStays(seToStaysChan, staysToSaveChan)

	// save and display stays
	go handleStays(staysToSaveChan, db)

	// start capturing packets and start goroutine to handle packets
	pcapIn, err := pcap.OpenLive("eth0", 400, true, pcap.BlockForever)
	// pcapIn, err := pcap.OpenOffline("20220329_RoboCap06.cap")
	check(err)
	packetSource := gopacket.NewPacketSource(pcapIn, pcapIn.LinkType())
	go handlePacket(packetSource.Packets(), srChan)

	// just do nothing, everything that happens now happens in non-main goroutines
	for {
		time.Sleep(100 * time.Second)
	}
}

// func takePictureRoutine(picSeIn <-chan SortEvent) {
// 	for {
// 		se := <-picSeIn

// 		// we could now check if the cow is flagged in some way...
// 		if se.Gate.Id == 3 || se.Gate.Id == 2 {
// 			takePicture(se)
// 		}
// 	}
// }

func SaveAndShowSE(seIn <-chan SortEvent, db *sql.DB, seToStaysChan chan<- SortEvent, seForPictures chan<- SortEvent) {
	for {
		se := <-seIn
		var objName string
		// if *takePictures {
		// 	seForPictures <- se
		objName = takePicture(se)
		if objName == "" {
			fmt.Println("Empty object name")
		} else {
			fmt.Println(objName)
		}
		// }
		insertSortEvent(se, objName, db)
		if *verboseFlag {
			ShowSortEvent(se)
		}
		seToStaysChan <- se
	}
}

func handleStays(stIn <-chan Stay, db *sql.DB) {
	for {
		st := <-stIn

		insertStay(st, db)
		if *verboseFlag {
			ShowStay(st)
		}
	}
}

func handlePacket(packetsChan <-chan gopacket.Packet, srChan chan<- SortEvent) {
	var pcapw *pcapgo.Writer
	// init pcap file
	if *savePcap {
		filename := fmt.Sprintf("raw_%v.pcap", time.Now().Format("2006-01-02_15-04-05"))
		f, err := os.Create(filename)
		check(err)
		defer f.Close()

		pcapw = pcapgo.NewWriter(f)
		if err := pcapw.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
			log.Fatalf("WriteFileHeader: %v", err)
		}
	}

	for {
		packet := <-packetsChan

		// save packet
		if *savePcap {
			err := pcapw.WritePacket(packet.Metadata().CaptureInfo, packet.Data())
			if err != nil {
				fmt.Println("ERROR:", err)
			}
		}

		if udp := packet.Layer(layers.LayerTypeUDP); udp != nil && len(udp.LayerPayload()) > 4 {
			if udp.LayerPayload()[0] == 0x00 && udp.LayerPayload()[1] == 0x05 && udp.LayerPayload()[2] == 0x01 && udp.LayerPayload()[3] == 0x0a {
				if len(udp.LayerPayload()) == 18 && packet.Metadata().CaptureLength == 60 {
					continue
				}
				if len(udp.LayerPayload()) == 222 && packet.Metadata().CaptureLength == 264 {
					se := decodeSortEvent(packet)
					srChan <- se

					continue
				}
			}
		}
	}
}

func printHex(data []byte) {
	for _, b := range data {
		if b == 0x0 {
			fmt.Printf("\033[38;5;%dm", 240)
			fmt.Printf("%02x ", b)
			fmt.Printf("\033[0m")
		} else {
			fmt.Printf("%02x ", b)
		}
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
